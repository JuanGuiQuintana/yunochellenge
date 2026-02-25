# Chargeback Intelligence API

API en Go para TiendaNube que ingesta chargebacks de múltiples procesadores, calcula prioridad (score 0-100), detecta patrones de fraude, y expone una query API.

**Métrica crítica:** chargebacks con score >= 70 deben notificarse en < 2 horas (hoy el proceso tarda hasta el día 5-6 de 7).

---

## Stack

- **Go 1.22** con `net/http` estándar (sin frameworks)
- **PostgreSQL 16** — única base de datos, sin Redis/Kafka/Elasticsearch
- **`pgx/v5`** directo — sin ORM. Maneja JSONB, arrays y UUIDs nativamente
- **`golang-migrate`** para migraciones SQL puras
- **`log/slog`** (stdlib) para logging estructurado
- **`errgroup`** para pattern detection paralelo
- Dinero en **`BIGINT` (centavos)** — nunca `FLOAT`

---

## Inicio rápido

### 1. Variables de entorno

```bash
cp .env.example .env
```

```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/chargeback_db
PORT=8080
APP_ENV=development
LOG_LEVEL=info
RUN_MIGRATIONS=true
DB_MAX_CONNS=25
DB_MIN_CONNS=5
```

### 2. Levantar PostgreSQL

```bash
make docker-up
```

### 3. Correr la API

```bash
make run
# Las migraciones corren automáticamente con RUN_MIGRATIONS=true
```

### 4. Cargar datos de prueba

```bash
make seed
```

---

## Comandos

| Comando | Descripción |
|---|---|
| `make run` | Corre la API (con hot-reload de migraciones) |
| `make build` | Compila a `bin/api` |
| `make docker-up` | Levanta PostgreSQL 16 |
| `make docker-down` | Detiene los contenedores |
| `make seed` | Carga 150+ chargebacks con patrones plantados |
| `make test` | Tests completos (unit + integration) |
| `make test-unit` | Solo tests sin DB (scoring, adapters, domain) |
| `make lint` | golangci-lint |

---

## Endpoints

### Ingestión

```
POST /api/v1/chargebacks/ingest/{processor_name}
  Header: X-Processor-Signature: <HMAC-SHA256>
  Body: payload específico del procesador (JSON)

  202 → { chargeback_id, risk_score, score_breakdown, flags }
  200 → mismo chargeback_id si ya existía (idempotente)
  401 → firma inválida
  429 → rate limit excedido
```

Procesadores soportados: `acquireco`, `payflow`, `globalpay`

### Query API

```
GET /api/v1/chargebacks                          filtros compuestos + paginación
GET /api/v1/chargebacks/summary                  counts por status + breakdown por deadline
GET /api/v1/chargebacks/{chargeback_id}          detalle + timeline de eventos
GET /api/v1/merchants/{merchant_id}/chargebacks  chargebacks de un merchant
GET /api/v1/merchants/{merchant_id}/stats        ratio actual + trend 30 días
```

### Filtros del listado

| Param | Tipo | Descripción |
|---|---|---|
| `merchant_id` | UUID | Filtrar por merchant |
| `score_min` | int | Score mínimo (0-100) |
| `deadline_hours` | int | Horas restantes al deadline |
| `deadline_before` | RFC3339 | Deadline anterior a esta fecha |
| `flags` | csv | `repeat_offender,merchant_hot_zone,...` |
| `flags_match` | `or`/`all` | Cualquier flag o todos |
| `reason_code` | string | Ej: `10.4`, `13.1` |
| `processor_name` | string | `acquireco`, `payflow`, `globalpay` |
| `status` | string | `open`, `under_review`, `won`, `lost`, `expired` |
| `currency` | string | `USD`, `BRL`, `MXN`, `ARS`, `COP` |
| `amount_min` | int | Monto mínimo en centavos |
| `amount_max` | int | Monto máximo en centavos |
| `sort_by` | string | `score`, `dispute_deadline`, `amount`, `notification_date` |
| `sort_order` | string | `asc`, `desc` |
| `page` | int | Default: 1 |
| `per_page` | int | Default: 25, max: 100 |

### Respuesta paginada

```json
{
  "data": [...],
  "pagination": {
    "total": 150,
    "page": 1,
    "per_page": 25,
    "total_pages": 6
  }
}
```

---

## Scoring (0-100 pts)

`Score = Time_Score + Amount_Score + ReasonCode_Score + Ratio_Score`

### Time Sensitivity (0-25 pts)

| Horas al deadline | Pts |
|---|---|
| <= 24h | 25 |
| 25-48h | 22 |
| 49-72h | 18 |
| 73-96h | 14 |
| 97-120h | 10 |
| 121-144h | 6 |
| > 144h | 2 |

### Amount en USD (0-25 pts)

| Monto | Pts |
|---|---|
| >= $1,000 | 25 |
| $500-$999 | 20 |
| $200-$499 | 15 |
| $100-$199 | 10 |
| $50-$99 | 6 |
| < $50 | 3 |

### Reason Code — risk_level (0-25 pts)

| Level | Pts |
|---|---|
| 5 | 25 |
| 4 | 20 |
| 3 | 14 |
| 2 | 8 |
| 1 | 4 |

### Merchant Ratio (0-25 pts)

| Ratio | Pts |
|---|---|
| >= 1.5% | 25 |
| 1.0-1.49% | 22 |
| 0.9-0.99% | 18 |
| 0.7-0.89% | 13 |
| 0.5-0.69% | 8 |
| 0.3-0.49% | 4 |
| < 0.3% | 1 |

**Ejemplo:** deadline en 20h + $800 + CNP (risk_level 4) + ratio 1.1% = 25+20+20+22 = **87** (Crítico)

Los flags **no modifican el score** — son una dimensión ortogonal para filtrado y contexto.

---

## Pattern Detection

Ejecutadas post-INSERT con `errgroup` (paralelas, solo COUNT queries sobre índices):

| Flag | Ventana | Umbral | Descripción |
|---|---|---|---|
| `repeat_offender` | 30 días | >= 3 | Mismo cardholder con múltiples chargebacks |
| `merchant_hot_zone` | 7 días | >= 5 | Merchant con spike de chargebacks |
| `suspicious_reason_clustering` | 14 días | >= 4 | Mismo merchant+reason_code repetido |

---

## Flujo de Ingestión (< 150ms)

```
[0ms]   Validar HMAC + rate limit (upsert atómico)
[10ms]  Deduplicación por processor_chargeback_id
[20ms]  Adapter.Normalize(raw) → ChargebackDTO + FindOrCreate cardholder
[40ms]  Enriquecer: merchant ratio cacheado + FX lookup (fallback a última tasa)
[50ms]  ScoringEngine: 4 componentes → risk_score + score_breakdown
[80ms]  ChargebackRepository.Insert
[120ms] PatternDetectionService: 3 COUNT queries paralelas → UpdateFlags
[150ms] Response 202
```

---

## Migraciones

| # | Tabla | Depende de |
|---|---|---|
| 001 | processors | — |
| 002 | merchants | — |
| 003 | cardholders | — |
| 004 | reason_codes | 001 |
| 005 | fx_rates | — |
| 006 | chargebacks | 001, 002, 003, 004 + todos los índices |
| 007 | chargeback_events | 006 |
| 008 | ingest_rate_limits | 001 |
| 009 | merchant_ratio_snapshots | 002 |

---

## Tests

```bash
# Unit tests (sin DB)
make test-unit

# Tests completos (requiere TEST_DATABASE_URL)
TEST_DATABASE_URL=postgresql://... make test
```

Cobertura:
- **Scoring**: cada tramo de cada tabla de lookup
- **Adapters**: fixtures JSON por procesador
- **Domain**: `CardFingerprint.Compute` determinístico y case-insensitive
- **HMAC**: firma válida/inválida
- **Integration**: concurrencia en `FindOrCreate`, rate limit atómico, pattern detection con seed
- **HTTP handlers**: 401, 404, 429, 202, 200 (idempotencia), paginación

---

## Estructura

```
cmd/api/main.go          # Entrada: config → DB → migrations → server
internal/
  config/                # Config + Load() fail-fast
  domain/                # Tipos puros sin dependencias externas
  processor/             # Adapters: acquireco, payflow, globalpay
  scoring/               # Engine + 4 componentes (time, amount, reason, ratio)
  repository/            # Acceso a DB con pgx/v5
  service/               # IngestService, PatternDetection, Enrichment, HMAC
  api/                   # Handlers, middleware, router, server
  db/                    # pgxpool setup
  migration/             # golang-migrate como librería
migrations/              # 001-009 .up.sql y .down.sql
scripts/
  seed.sql               # 150+ chargebacks con patrones plantados
  demo.sh                # curl commands de las 5 queries principales
```
