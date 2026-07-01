package session

// NormalizeStatus maps common CI status strings to pipepush statuses, so
// CI-native values like "passed"/"failed" work directly.
func NormalizeStatus(s string) string {
	switch s {
	case "success", "passed", "ok", "succeeded":
		return "success"
	case "failure", "failed", "error", "broken":
		return "failure"
	case "cancelled", "canceled", "aborted":
		return "cancelled"
	case "running", "started", "in_progress", "pending":
		return "running"
	case "skipped":
		return "skipped"
	default:
		return s
	}
}
