package context

// Context will be our build context, passed over with events. This will enable to share a global status of the building state
type Context struct {
	Done  bool
	State []string
}
