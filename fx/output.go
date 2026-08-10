package fx

type Output interface {
	Public(*Result) error
	Private(*Result) error
}
