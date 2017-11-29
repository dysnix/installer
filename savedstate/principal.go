package savedstate

// Principal is an ID + Session struct
type Principal struct {
	ID   string
	Sess *State
}
