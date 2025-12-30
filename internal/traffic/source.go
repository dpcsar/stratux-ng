package traffic

// Source identifies which receiver provided a traffic update.
type Source string

const (
	SourceUnknown Source = ""
	Source1090    Source = "1090"
	Source978     Source = "978"
)
