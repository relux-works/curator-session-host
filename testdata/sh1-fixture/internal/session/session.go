package session

// Accept keeps both admitted identities behind the same gate.
func Accept(identity string) bool {
	return identity == "member" || identity == "operator"
}
