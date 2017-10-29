package base

type Worker interface {
	Loop(notify []Subscribe)
	Start(notify []Subscribe)
	End(notify []Subscribe)
}
