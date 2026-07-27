package adminops

import "github.com/storesace-cv/bwb-modulo-fiscal/internal/notify/smtp"

// AlertDigestLines maps ops alerts to SMTP digest lines (allowlisted fields only).
func AlertDigestLines(alerts []OpsAlert) []smtp.AlertLine {
	out := make([]smtp.AlertLine, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, smtp.AlertLine{
			Code:     a.Code,
			Severity: string(a.Severity),
			Message:  a.Message,
		})
	}
	return out
}

// AlertCodes returns allowlisted alert codes in order (for sanitized API responses).
func AlertCodes(alerts []OpsAlert) []string {
	out := make([]string, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, a.Code)
	}
	return out
}
