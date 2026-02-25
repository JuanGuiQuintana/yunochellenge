# CLAUDE.md — Chargeback Intelligence API

## Proyecto

API en Go para TiendaNube que ingesta chargebacks de múltiples procesadores, calcula prioridad (score 0-100), detecta patrones de fraude, y expone una query API. Challenge: The Chargeback Tsunami.

**Métrica crítica:** chargebacks con score >= 70 deben notificarse en < 2 horas (hoy el proceso tarda hasta el día 5-6 de 7).

---

## Stack y Restricciones

- **Solo PostgreSQL** — sin Redis, Kafka, Elasticsearch, MongoDB. 300 chargebacks/día no justifica más infraestructura.
- **Go 1.22** con `net/http` estándar — sin frameworks (Gin, Echo, Chi).
- **`pgx/v5` directo** — sin ORM (GORM), sin `sqlx`. pgx maneja JSONB, arrays, y UUIDs nativamente.
- **`golang-migrate`** para migraciones SQL puras.
- **`log/slog`** (stdlib) para logging estructurado.
- **`errgroup`** (`golang.org/x/sync`) para pattern detection paralelo.
- Dinero en **`BIGINT` (centavos)** — nunca `FLOAT`.

---

## Estructura de Directorios

```
cmd/api/main.go                         # Entrada: config → DB → migrations → server
internal/
  config/config.go                      # Config struct + Load() desde env, fail-fast
  domain/                               # Tipos puros, sin dependencias externas
    chargeback.go                       # Chargeback, ChargebackStatus, Flag, ChargebackSummary
    merchant.go                         # Merchant, MerchantStats
    cardholder.go                       # Cardholder, CardFingerprint (value object con SHA-256)
    processor.go                        # Processor
    reasoncode.go                       # ReasonCode, RiskLevel
    scoring.go                          # ScoreBreakdown con MarshalJSON/UnmarshalJSON
    filter.go                           # ChargebackFilter (todos los query params como punteros)
    fxrate.go                           # FxRate
    errors.go                           # ErrNotFound, ErrDuplicate, ErrInvalidInput, ErrRateLimitExceeded, ErrUnauthorized
  processor/
    adapter.go                          # Interfaz ProcessorAdapter + ChargebackDTO
    acquireco/adapter.go
    payflow/adapter.go
    globalpay/adapter.go
  scoring/
    engine.go                           # ScoringEngine + ScoringInput + ComponentResult + interfaz ScoringComponent
    time_sensitivity.go                 # 0-25 pts por horas al deadline
    amount.go                           # 0-25 pts por monto USD
    reason_code.go                      # 0-25 pts por risk_level
    merchant_ratio.go                   # 0-25 pts por ratio de chargebacks
  repository/
    chargeback.go                       # Insert, FindByID, FindByProcessorID, List, UpdateFlags, Summary
    merchant.go                         # FindByID, UpdateRatioCache, Stats
    cardholder.go                       # FindOrCreate (upsert por card_fingerprint)
    processor.go                        # FindByName (con cache en memoria)
    reasoncode.go                       # FindByRawCode
    fxrate.go                           # GetRate con fallback a última tasa conocida
    ratelimit.go                        # CheckAndIncrement (upsert atómico)
  service/
    ingest.go                           # IngestService: orquesta el flujo de 8 pasos
    pattern.go                          # PatternDetectionService: 3 COUNT queries con errgroup
    enrichment.go                       # FX lookup + short-circuit para USD
    hmac.go                             # Validate(secret, signature, body) con hmac.Equal
  api/
    server.go                           # http.Server con timeouts
    router.go                           # Registrar /chargebacks/summary ANTES de /chargebacks/{id}
    middleware/
      requestid.go                      # X-Request-ID con type contextKey string (key privada)
      logging.go                        # slog
      recovery.go                       # panic → 500
    handler/
      ingest.go                         # POST /api/v1/chargebacks/ingest/{processor_name}
      chargeback.go                     # GET /chargebacks, GET /chargebacks/{id}, GET /chargebacks/summary
      merchant.go                       # GET /merchants/{id}/chargebacks, GET /merchants/{id}/stats
      processor.go                      # GET /processors (stretch)
    response/
      json.go                           # WriteJSON(w, status, v), WriteError(w, status, err)
      pagination.go                     # PaginatedResponse[T] genérico
  db/postgres.go                        # pgxpool.Pool setup + pgxuuid.Register
  migration/runner.go                   # golang-migrate integrado como librería
migrations/                             # 001 a 009 .up.sql y .down.sql
scripts/
  seed.sql                              # 150+ chargebacks con patrones plantados
  demo.sh                               # curl commands de las 5 queries principales
docker-compose.yml                      # PostgreSQL 16
Makefile
.env.example
README.md
```

---

## Endpoints

### Ingestión
```
POST /api/v1/chargebacks/ingest/{processor_name}
  Header: X-Processor-Signature (HMAC-SHA256)
  Response 202: { chargeback_id, risk_score, score_breakdown, flags }
  Idempotente: mismo processor_chargeback_id+processor_name → 200 con registro existente
```

### Query API
```
GET /api/v1/chargebacks                          filtros compuestos + paginación
GET /api/v1/chargebacks/summary                  counts por status, breakdown por deadline
GET /api/v1/chargebacks/{chargeback_id}          detalle + timeline de eventos
GET /api/v1/merchants/{merchant_id}/chargebacks  alias ergonómico
GET /api/v1/merchants/{merchant_id}/stats        ratio actual + trend 30 días
```

### Query params del listado
`merchant_id`, `score_min`, `deadline_hours`, `deadline_before`, `flags` (csv), `flags_match` (or/all), `reason_code`, `processor_name`, `status`, `currency`, `amount_min`, `amount_max`, `sort_by` (score/dispute_deadline/amount/notification_date), `sort_order` (asc/desc), `page` (default 1), `per_page` (default 25, max 100)

### Respuesta paginada (todos los listados)
```json
{ "data": [...], "pagination": { "total": 150, "page": 1, "per_page": 25, "total_pages": 6 } }
```

### Stretch
```
GET  /api/v1/analytics/trends
GET  /api/v1/analytics/reason-codes
GET  /api/v1/processors
POST /api/v1/chargebacks/{id}/evidence
```

---

## Reglas de Scoring

`Score = Time_Score + Amount_Score + ReasonCode_Score + Ratio_Score` (cada uno 0-25)

### Time Sensitivity
| Horas al deadline | Pts |
|---|---|
| <= 24h | 25 |
| 25-48h | 22 |
| 49-72h | 18 |
| 73-96h | 14 |
| 97-120h | 10 |
| 121-144h | 6 |
| > 144h | 2 |

### Amount (en USD)
| Monto | Pts |
|---|---|
| >= $1,000 | 25 |
| $500-$999 | 20 |
| $200-$499 | 15 |
| $100-$199 | 10 |
| $50-$99 | 6 |
| < $50 | 3 |

### Reason Code (por risk_level)
| Level | Pts |
|---|---|
| 5 | 25 |
| 4 | 20 |
| 3 | 14 |
| 2 | 8 |
| 1 | 4 |

### Merchant Ratio
| Ratio | Pts |
|---|---|
| >= 1.5% | 25 |
| 1.0-1.49% | 22 |
| 0.9-0.99% | 18 |
| 0.7-0.89% | 13 |
| 0.5-0.69% | 8 |
| 0.3-0.49% | 4 |
| < 0.3% | 1 |

**Caso de referencia:** deadline en 20h + $800 + CNP (risk_level 4) + ratio 1.1% = 25+20+20+22 = **87** (Crítico)

**Los flags NO modifican el score** — son dimensión ortogonal para filtrado/contexto.

---

## Pattern Detection

Todas como COUNT queries SQL sobre índices compuestos, ejecutadas **después del INSERT** con `errgroup`:

| Flag | Query | Ventana | Umbral | Flag aparece en... |
|---|---|---|---|---|
| `repeat_offender` | COUNT por `cardholder_id` | 30 días | >= 3 | 3er chargeback |
| `merchant_hot_zone` | COUNT por `merchant_id` | 7 días | >= 5 | 5to chargeback |
| `suspicious_reason_clustering` | COUNT por `merchant_id + reason_code_id` | 14 días | >= 4 | 4to chargeback |

Índices requeridos en tabla `chargebacks`:
- `(cardholder_id, notification_date)` — repeat_offender
- `(merchant_id, notification_date)` — hot_zone
- `(merchant_id, reason_code_id, notification_date)` — clustering
- `(merchant_id, dispute_deadline)` — queries de urgencia
- `(merchant_id, risk_score DESC)` — queries de alta prioridad

---

## Flujo de Ingestión (< 150ms)

```
[0ms]   Validar HMAC + rate limit (upsert atómico en IngestRateLimits)
[10ms]  Deduplicación por processor_chargeback_id
[20ms]  Adapter.Normalize(raw) → ChargebackDTO; FindOrCreate cardholder por card_fingerprint
[40ms]  Enriquecer: merchant ratio (columna cacheada) + FX lookup (fallback a última tasa)
[50ms]  ScoringEngine: 4 componentes → risk_score + score_breakdown
[80ms]  ChargebackRepository.Insert
[120ms] PatternDetectionService: 3 COUNT queries con errgroup → UpdateFlags
[150ms] Response 202
```

---

## Decisiones Técnicas Clave

### Errores de dominio
```go
// domain/errors.go — mapeo a HTTP en los handlers con errors.Is()
var (
    ErrNotFound          error // 404
    ErrDuplicate         error // 200 (idempotencia)
    ErrInvalidInput      error // 400
    ErrRateLimitExceeded error // 429
    ErrUnauthorized      error // 401
)
```

### Filtros dinámicos — nunca fmt.Sprintf con valores
```go
// Construir con placeholders numerados $N para prevenir SQL injection
conditions = append(conditions, fmt.Sprintf("risk_score >= $%d", n))
args = append(args, *filter.ScoreMin)
```

### Rate limit — upsert atómico
```sql
INSERT INTO ingest_rate_limits (processor_id, window_minute, request_count)
VALUES ($1, date_trunc('minute', NOW()), 1)
ON CONFLICT (processor_id, window_minute)
DO UPDATE SET request_count = ingest_rate_limits.request_count + 1
RETURNING request_count
```

### FX fallback
```
1. SELECT rate WHERE currency=$1 AND date=$2
2. Si no existe: SELECT rate WHERE currency=$1 ORDER BY date DESC LIMIT 1
3. Si no existe: amount_usd=null, fx_pending=true en score_breakdown (NO bloquear)
4. USD: retornar 1.0 sin query
```

### CardFingerprint
```go
// SHA-256 de BIN|last4|NAME_NORMALIZADO — nunca guardar nombre en texto plano
func Compute(bin, last4, name string) CardFingerprint {
    normalized := strings.ToUpper(strings.TrimSpace(name))
    hash := sha256.Sum256([]byte(bin + "|" + last4 + "|" + normalized))
    return CardFingerprint(hex.EncodeToString(hash[:]))
}
```

### Pattern detection paralelo
```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { /* repeat_offender COUNT */ return nil })
g.Go(func() error { /* hot_zone COUNT */ return nil })
g.Go(func() error { /* clustering COUNT */ return nil })
if err := g.Wait(); err != nil { return nil, err }
```

### Router ordering — crítico
```go
// router.go: summary DEBE registrarse antes de {chargeback_id}
mux.HandleFunc("GET /api/v1/chargebacks/summary", h.Summary)
mux.HandleFunc("GET /api/v1/chargebacks/{chargeback_id}", h.GetByID)
```

### UUID en pgx
```go
// db/postgres.go — AfterConnect
pgxuuid.Register(conn.TypeMap())
// Generar en Go, no en DB
id := uuid.New()
```

---

## Migraciones (orden de dependencias FK)

| # | Tabla | Depende de |
|---|---|---|
| 001 | processors | — |
| 002 | merchants | — |
| 003 | cardholders | — |
| 004 | reason_codes | processors |
| 005 | fx_rates | — |
| 006 | chargebacks | 001,002,003,004 — incluye todos los índices compuestos |
| 007 | chargeback_events | 006 |
| 008 | ingest_rate_limits | 001 |
| 009 | merchant_ratio_snapshots | 002 |

DDL crítico en `006_create_chargebacks`:
- `chargeback_id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `status TEXT NOT NULL CHECK (status IN ('open','under_review','won','lost','expired'))`
- `flags TEXT[] NOT NULL DEFAULT '{}'`
- `amount BIGINT NOT NULL` (centavos)
- `amount_usd NUMERIC(12,2)`
- `score_breakdown JSONB NOT NULL DEFAULT '{}'`
- `raw_payload JSONB NOT NULL`

---

## Variables de Entorno

```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/chargeback_db
PORT=8080
APP_ENV=development          # "development" | "production"
LOG_LEVEL=info
RUN_MIGRATIONS=true          # false en producción
DB_MAX_CONNS=25
DB_MIN_CONNS=5
```

`Load()` falla inmediatamente si `DATABASE_URL` está vacío.

---

## Testing

### Unit (sin DB)
- `scoring/*`: cada tramo de cada tabla de lookup — incluir el caso de referencia 87
- `processor/*/adapter`: fixtures JSON por procesador → DTO correcto
- `domain/cardholder`: `Compute` es determinístico y case-insensitive
- `service/hmac`: firma válida/inválida con `hmac.Equal`

### Integration (con DB real via `TEST_DATABASE_URL`)
- `TestMain` crea schema → corre migraciones → cleanup al terminar
- `CardholderRepository.FindOrCreate`: 10 goroutines mismo fingerprint → 1 registro
- `RateLimitRepository.CheckAndIncrement`: límite respetado, ventana nueva resetea
- `PatternDetectionService`: seed con patrones conocidos → flags correctos
- `IngestService.Ingest`: flujo completo + idempotencia (2da llamada → mismo chargeback_id)

### HTTP Handlers (`httptest.NewRecorder()`)
- HMAC inválida → 401
- Procesador desconocido → 404
- Rate limit → 429
- Ingest exitoso → 202 con schema correcto
- Ingest duplicado → 200
- Paginación correcta (total_pages)

### Mocks — manuales (no mockgen)
```go
type mockChargebackRepository struct {
    InsertFn func(ctx context.Context, cb domain.Chargeback) error
}
func (m *mockChargebackRepository) Insert(ctx context.Context, cb domain.Chargeback) error {
    return m.InsertFn(ctx, cb)
}
```

---

## Test Data (seed.sql)

Requisitos mínimos:
- 150+ chargebacks
- 20+ merchants únicos
- 3 procesadores: AcquireCo, PayFlow, GlobalPay
- 5+ reason codes (10.4, 13.1, 13.2, etc.)
- 60 días de historia con deadlines variados
- Monedas: USD, BRL, MXN, ARS, COP
- Montos: $10 a $2,500
- **Patrones plantados verificables:**
  - 3 cardholders con >= 3 chargebacks en 30 días (repeat_offender)
  - 2 merchants con >= 5 chargebacks en 7 días (merchant_hot_zone)
  - 1 merchant con ratio > 1% (alto ratio)

---

## Orden de Implementación

```
Fase 1: config + db + migrations (arranca y conecta)
Fase 2: domain types y value objects (compilan sin errores)
Fase 3/4/5: repositories + adapters + scoring (paralelas entre sí)
Fase 6: services (requiere 3+4+5)
Fase 7: API HTTP (requiere 6)
Fase 8: seed + demo.sh + README
```
