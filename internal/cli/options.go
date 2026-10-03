package cli

// WriteOption tweaks an Inject or Update call beyond its required
// arguments - variadic so adding one never touches existing callers.
type WriteOption func(*writeOptions)

type writeOptions struct {
	// tags is nil when unset. On Update a non-nil empty slice clears the
	// object's tags; on Inject empty and nil are the same.
	tags *[]string
	// usedBy is nil when unset. On Update a non-nil empty slice clears the
	// object's consumers.
	usedBy *[]string
}

func newWriteOptions(opts []WriteOption) writeOptions {
	var w writeOptions
	for _, o := range opts {
		o(&w)
	}

	return w
}

// tagsOrNil is the tags to send on create: nothing, unless some were given.
func (w writeOptions) tagsOrNil() []string {
	if w.tags == nil {
		return nil
	}

	return *w.tags
}

// WithTags sets an object's tags: on Inject the initial ones, on Update a
// full replacement. Pass an empty, non-nil slice to Update to clear them.
func WithTags(tags []string) WriteOption {
	return func(w *writeOptions) { w.tags = &tags }
}

// WithUsedBy replaces an object's consumers on Update. Pass an empty,
// non-nil slice to clear them. Inject takes its consumers as an argument.
func WithUsedBy(usedBy []string) WriteOption {
	return func(w *writeOptions) { w.usedBy = &usedBy }
}
