package base

type Worker interface {
	Loop(notify []Subscribe)
}
