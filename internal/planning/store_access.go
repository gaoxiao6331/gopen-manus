package planning

// InternalPlan exposes the raw plan for internal flow usage.
// This is intentionally minimal to keep tool APIs out of the Go port.
func (s *Store) InternalPlan(planID string) (*Plan, bool) {
	if s == nil {
		return nil, false
	}
	plan, ok := s.plans[planID]
	return plan, ok
}
