package types

type Vote int32

const (
	// TODO HV2: should we have an UNSPECIFIED value 0 tag as suggested here: https://protobuf.dev/programming-guides/dos-donts/#unspecified-enum ?
	// currently the zero value of the enum would be interpreted as VOTE_SKIP
	Vote_VOTE_SKIP Vote = 0
	Vote_VOTE_YES  Vote = 1
	Vote_VOTE_NO   Vote = 2
)

func (v Vote) String() string {

	switch v {
	case Vote_VOTE_YES:
		return "YES"
	case Vote_VOTE_NO:
		return "NO"
	case Vote_VOTE_SKIP:
		return "SKIP"

	}

	return ""
}
