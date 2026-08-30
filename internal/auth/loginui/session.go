package loginui

// State is the client-safe view of an in-progress login: which factors have
// been verified so far. ZITADEL leaves it to the calling client to decide
// when a session is sufficient to authenticate.
type State struct {
	LoginName        string `json:"loginName,omitempty"`
	PasswordVerified bool   `json:"passwordVerified"`
	// NeedsTOTP is set once the login name is known and the account has TOTP
	// enrolled; the frontend must then also collect and submit a TOTP code
	// before the session can be sufficient.
	NeedsTOTP    bool `json:"needsTotp"`
	TotpVerified bool `json:"totpVerified"`
}

// Sufficient reports whether State has enough verified factors to finalize
// the login: a verified password, plus a verified TOTP code if the account
// has TOTP enrolled.
func (s State) Sufficient() bool {
	if s.LoginName == "" || !s.PasswordVerified {
		return false
	}
	return !s.NeedsTOTP || s.TotpVerified
}
