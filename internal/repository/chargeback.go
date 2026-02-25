package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgChargebackRepository struct {
	db *pgxpool.Pool
}

// NewChargebackRepository returns a ChargebackRepository backed by PostgreSQL.
func NewChargebackRepository(db *pgxpool.Pool) ChargebackRepository {
	return &pgChargebackRepository{db: db}
}

func (r *pgChargebackRepository) Insert(ctx context.Context, cb domain.Chargeback) error {
	breakdownBytes, err := cb.ScoreBreakdown.ToBytes()
	if err != nil {
		return fmt.Errorf("chargeback.Insert marshal breakdown: %w", err)
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO chargebacks (
		    chargeback_id, processor_chargeback_id, processor_id, merchant_id,
		    transaction_id, cardholder_id, reason_code_id,
		    amount, currency, amount_usd,
		    transaction_date, notification_date, dispute_deadline,
		    status, raw_payload, risk_score, score_breakdown, flags
		) VALUES (
		    $1, $2, $3, $4, $5, $6, $7,
		    $8, $9, $10,
		    $11, $12, $13,
		    $14, $15, $16, $17, $18
		)`,
		cb.ChargebackID, cb.ProcessorChargebackID, cb.ProcessorID, cb.MerchantID,
		cb.TransactionID, cb.CardholderID, cb.ReasonCodeID,
		cb.Amount, cb.Currency, cb.AmountUSD,
		cb.TransactionDate, cb.NotificationDate, cb.DisputeDeadline,
		string(cb.Status), cb.RawPayload, cb.RiskScore, breakdownBytes, cb.Flags,
	)
	if err != nil {
		return fmt.Errorf("chargeback.Insert: %w", err)
	}
	return nil
}

func (r *pgChargebackRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Chargeback, error) {
	row := r.db.QueryRow(ctx,
		`SELECT chargeback_id, processor_chargeback_id, processor_id, merchant_id,
		        transaction_id, cardholder_id, reason_code_id,
		        amount, currency, amount_usd,
		        transaction_date, notification_date, dispute_deadline,
		        status, raw_payload, risk_score, score_breakdown, flags,
		        created_at, updated_at
		 FROM chargebacks WHERE chargeback_id = $1`, id)

	cb, err := scanChargeback(row)
	if err == pgx.ErrNoRows {
		return domain.Chargeback{}, fmt.Errorf("chargeback %s: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Chargeback{}, fmt.Errorf("chargeback.FindByID: %w", err)
	}
	return cb, nil
}

func (r *pgChargebackRepository) FindByProcessorID(ctx context.Context, processorName, processorCBID string) (domain.Chargeback, bool, error) {
	row := r.db.QueryRow(ctx,
		`SELECT c.chargeback_id, c.processor_chargeback_id, c.processor_id, c.merchant_id,
		        c.transaction_id, c.cardholder_id, c.reason_code_id,
		        c.amount, c.currency, c.amount_usd,
		        c.transaction_date, c.notification_date, c.dispute_deadline,
		        c.status, c.raw_payload, c.risk_score, c.score_breakdown, c.flags,
		        c.created_at, c.updated_at
		 FROM chargebacks c
		 JOIN processors p ON p.processor_id = c.processor_id
		 WHERE p.name = $1 AND c.processor_chargeback_id = $2`,
		processorName, processorCBID,
	)

	cb, err := scanChargeback(row)
	if err == pgx.ErrNoRows {
		return domain.Chargeback{}, false, nil
	}
	if err != nil {
		return domain.Chargeback{}, false, fmt.Errorf("chargeback.FindByProcessorID: %w", err)
	}
	return cb, true, nil
}

// List builds a dynamic WHERE clause from ChargebackFilter using numbered placeholders
// to prevent SQL injection, then executes a count query and a paginated select.
func (r *pgChargebackRepository) List(ctx context.Context, filter domain.ChargebackFilter) ([]domain.Chargeback, int, error) {
	selectQuery, countQuery, args, filterArgCount := buildListQuery(filter)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args[:filterArgCount]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("chargeback.List count: %w", err)
	}

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("chargeback.List query: %w", err)
	}
	defer rows.Close()

	var chargebacks []domain.Chargeback
	for rows.Next() {
		cb, err := scanChargebackRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("chargeback.List scan: %w", err)
		}
		chargebacks = append(chargebacks, cb)
	}
	return chargebacks, total, rows.Err()
}

func (r *pgChargebackRepository) UpdateFlags(ctx context.Context, id uuid.UUID, flags []string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE chargebacks SET flags = $2, updated_at = NOW() WHERE chargeback_id = $1`,
		id, flags,
	)
	if err != nil {
		return fmt.Errorf("chargeback.UpdateFlags: %w", err)
	}
	return nil
}

func (r *pgChargebackRepository) Summary(ctx context.Context, merchantID *uuid.UUID) (domain.ChargebackSummary, error) {
	args := []any{}
	whereClause := ""
	if merchantID != nil {
		whereClause = "WHERE merchant_id = $1"
		args = append(args, *merchantID)
	}

	var s domain.ChargebackSummary
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT
		    COUNT(*) FILTER (WHERE status = 'open') AS total_open,
		    COUNT(*) FILTER (WHERE status = 'under_review') AS total_responded,
		    COUNT(*) FILTER (WHERE status = 'won') AS total_won,
		    COUNT(*) FILTER (WHERE status = 'lost') AS total_lost,
		    COUNT(*) FILTER (WHERE status = 'expired') AS total_expired,
		    COALESCE(AVG(risk_score), 0) AS avg_risk_score,
		    COUNT(*) FILTER (WHERE status = 'open' AND dispute_deadline <= NOW() + INTERVAL '24 hours') AS expiring_24h,
		    COUNT(*) FILTER (WHERE status = 'open' AND dispute_deadline <= NOW() + INTERVAL '48 hours') AS expiring_48h,
		    COUNT(*) FILTER (WHERE status = 'open' AND dispute_deadline <= NOW() + INTERVAL '72 hours') AS expiring_72h
		FROM chargebacks %s`, whereClause), args...,
	).Scan(&s.TotalOpen, &s.TotalResponded, &s.TotalWon, &s.TotalLost, &s.TotalExpired,
		&s.AvgRiskScore, &s.ExpiringIn24h, &s.ExpiringIn48h, &s.ExpiringIn72h)

	if err != nil {
		return domain.ChargebackSummary{}, fmt.Errorf("chargeback.Summary: %w", err)
	}
	return s, nil
}

// --- scan helpers ---

// scannable is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query iteration),
// allowing a single scan function to serve both call sites.
type scannable interface {
	Scan(dest ...any) error
}

func scanChargeback(row scannable) (domain.Chargeback, error) {
	var cb domain.Chargeback
	var statusStr string
	var breakdownBytes []byte

	err := row.Scan(
		&cb.ChargebackID, &cb.ProcessorChargebackID, &cb.ProcessorID, &cb.MerchantID,
		&cb.TransactionID, &cb.CardholderID, &cb.ReasonCodeID,
		&cb.Amount, &cb.Currency, &cb.AmountUSD,
		&cb.TransactionDate, &cb.NotificationDate, &cb.DisputeDeadline,
		&statusStr, &cb.RawPayload, &cb.RiskScore, &breakdownBytes, &cb.Flags,
		&cb.CreatedAt, &cb.UpdatedAt,
	)
	if err != nil {
		return domain.Chargeback{}, err
	}

	cb.Status = domain.ChargebackStatus(statusStr)

	if len(breakdownBytes) > 0 {
		cb.ScoreBreakdown, err = domain.ScoreBreakdownFromBytes(breakdownBytes)
		if err != nil {
			return domain.Chargeback{}, fmt.Errorf("unmarshal score_breakdown: %w", err)
		}
	}

	return cb, nil
}

func scanChargebackRow(rows pgx.Rows) (domain.Chargeback, error) {
	var cb domain.Chargeback
	var statusStr string
	var breakdownBytes []byte

	err := rows.Scan(
		&cb.ChargebackID, &cb.ProcessorChargebackID, &cb.ProcessorID, &cb.MerchantID,
		&cb.TransactionID, &cb.CardholderID, &cb.ReasonCodeID,
		&cb.Amount, &cb.Currency, &cb.AmountUSD,
		&cb.TransactionDate, &cb.NotificationDate, &cb.DisputeDeadline,
		&statusStr, &cb.RawPayload, &cb.RiskScore, &breakdownBytes, &cb.Flags,
		&cb.CreatedAt, &cb.UpdatedAt,
	)
	if err != nil {
		return domain.Chargeback{}, err
	}

	cb.Status = domain.ChargebackStatus(statusStr)

	if len(breakdownBytes) > 0 {
		cb.ScoreBreakdown, err = domain.ScoreBreakdownFromBytes(breakdownBytes)
		if err != nil {
			return domain.Chargeback{}, fmt.Errorf("unmarshal score_breakdown: %w", err)
		}
	}
	return cb, nil
}

// buildListQuery constructs a SELECT and a COUNT query from the given filter.
// It returns:
//   - selectQuery: paginated SELECT with ORDER BY, LIMIT, OFFSET
//   - countQuery:  SELECT COUNT(*) with identical WHERE clause (no pagination args)
//   - args: combined arg slice; the first filterArgCount elements belong to both
//     queries and the final two (LIMIT, OFFSET) belong to selectQuery only
//   - filterArgCount: number of args consumed by the WHERE clause alone
//
// Sort columns are validated against a whitelist — user-supplied values from SortBy
// are never interpolated directly into the query string.
func buildListQuery(filter domain.ChargebackFilter) (selectQuery, countQuery string, args []any, filterArgCount int) {
	conditions := []string{"1=1"}
	args = []any{}
	n := 1

	if filter.MerchantID != nil {
		conditions = append(conditions, fmt.Sprintf("c.merchant_id = $%d", n))
		args = append(args, *filter.MerchantID)
		n++
	}
	if filter.ScoreMin != nil {
		conditions = append(conditions, fmt.Sprintf("c.risk_score >= $%d", n))
		args = append(args, *filter.ScoreMin)
		n++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", n))
		args = append(args, string(*filter.Status))
		n++
	}
	if filter.Currency != nil {
		conditions = append(conditions, fmt.Sprintf("c.currency = $%d", n))
		args = append(args, *filter.Currency)
		n++
	}
	if filter.AmountMin != nil {
		conditions = append(conditions, fmt.Sprintf("c.amount >= $%d", n))
		args = append(args, *filter.AmountMin)
		n++
	}
	if filter.AmountMax != nil {
		conditions = append(conditions, fmt.Sprintf("c.amount <= $%d", n))
		args = append(args, *filter.AmountMax)
		n++
	}
	if filter.DeadlineBefore != nil {
		conditions = append(conditions, fmt.Sprintf("c.dispute_deadline <= $%d", n))
		args = append(args, *filter.DeadlineBefore)
		n++
	}
	if filter.DeadlineHours != nil {
		conditions = append(conditions, fmt.Sprintf("c.dispute_deadline <= NOW() + ($%d * INTERVAL '1 hour')", n))
		args = append(args, *filter.DeadlineHours)
		n++
	}
	if filter.ReasonCode != nil {
		conditions = append(conditions, fmt.Sprintf("rc.normalized_code = $%d", n))
		args = append(args, *filter.ReasonCode)
		n++
	}
	if filter.ProcessorName != nil {
		conditions = append(conditions, fmt.Sprintf("p.name = $%d", n))
		args = append(args, *filter.ProcessorName)
		n++
	}
	if len(filter.Flags) > 0 {
		if filter.FlagsMatch == domain.FlagsMatchAll {
			// Every flag must be present in the array.
			for _, flag := range filter.Flags {
				conditions = append(conditions, fmt.Sprintf("$%d = ANY(c.flags)", n))
				args = append(args, flag)
				n++
			}
		} else {
			// Any flag matches — the && (overlaps) operator checks for intersection.
			conditions = append(conditions, fmt.Sprintf("c.flags && $%d::text[]", n))
			args = append(args, filter.Flags)
			n++
		}
	}

	whereSQL := strings.Join(conditions, " AND ")

	// Whitelist sort columns to prevent SQL injection from untrusted SortBy values.
	sortCol := "c.dispute_deadline"
	switch filter.SortBy {
	case domain.SortByScore:
		sortCol = "c.risk_score"
	case domain.SortByAmount:
		sortCol = "c.amount"
	case domain.SortByNotificationDate:
		sortCol = "c.notification_date"
	}
	sortDir := "ASC"
	if filter.SortOrder == domain.SortDesc {
		sortDir = "DESC"
	}

	joins := `FROM chargebacks c
		LEFT JOIN processors p ON p.processor_id = c.processor_id
		LEFT JOIN reason_codes rc ON rc.reason_code_id = c.reason_code_id`

	selectCols := `c.chargeback_id, c.processor_chargeback_id, c.processor_id, c.merchant_id,
		c.transaction_id, c.cardholder_id, c.reason_code_id,
		c.amount, c.currency, c.amount_usd,
		c.transaction_date, c.notification_date, c.dispute_deadline,
		c.status, c.raw_payload, c.risk_score, c.score_breakdown, c.flags,
		c.created_at, c.updated_at`

	filterArgCount = n - 1

	countQuery = fmt.Sprintf(`SELECT COUNT(*) %s WHERE %s`, joins, whereSQL)

	// Append LIMIT and OFFSET as the final two args so they don't shift filter $N positions.
	args = append(args, filter.PerPage, filter.Offset())
	selectQuery = fmt.Sprintf(
		`SELECT %s %s WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		selectCols, joins, whereSQL, sortCol, sortDir, n, n+1,
	)

	return selectQuery, countQuery, args, filterArgCount
}
