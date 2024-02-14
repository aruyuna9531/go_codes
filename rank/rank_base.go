package rank

type RankerBase[K uint64, V int64] struct {
	RankerId   K
	Value      V
	UpdateTime int64
	rankPtr    *RankBase[K]
}

type RankBase[K uint64] struct {
	Rankers // 有序map
}
