package update

// hasAllFeatures reports whether have contains every feature in need (AND
// match). An empty need is always satisfied.
func hasAllFeatures(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(have))
	for _, f := range have {
		set[f] = struct{}{}
	}
	for _, n := range need {
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}
