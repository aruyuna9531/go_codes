package rank

import (
	"fmt"
	"github.com/aruyuna9531/skiplist"
)

type SortableInt interface {
	int | int32 | int64 | uint | uint32 | uint64
}

type IRanker[K comparable] interface {
	isRanker()
	Key() K
	Less(skiplist.ISkiplistElement[K]) bool
}

type Ranker[K comparable, V SortableInt] struct {
	RankerId   K
	Value      V
	UpdateTime int64
	rankPtr    *RankBase[K, V]
}

func (r *Ranker[K, V]) isRanker() {

}

func (r *Ranker[K, V]) Key() K {
	return r.RankerId
}

func (r *Ranker[K, V]) Less(i skiplist.ISkiplistElement[K]) bool {
	ii, ok := i.(*Ranker[K, V])
	if !ok {
		panic("RankerBase::Less error: types different")
	}
	if r.Value > ii.Value {
		return true
	}
	if r.Value < ii.Value {
		return false
	}
	return r.UpdateTime < ii.UpdateTime
}

// GetRank 获得这个节点在所在排行榜上的排名（rankPtr是用在这儿的）不需要大费周章地在业务层找具体排行榜实例的位置避免出差错。（用在使用Range批量捞起区间内ranker，要取它们的实际排名写到邮件里）
func (r *Ranker[K, V]) GetRank() (ret int32, err error) {
	if r.rankPtr == nil {
		return 0, fmt.Errorf("Ranker::GetRank error: rankPtr = nil")
	}
	return r.rankPtr.GetRank(r.Key())
}

type RankBase[K comparable, V SortableInt] struct {
	rankMain *skiplist.SkipList[K]
	dict     map[K]V
}

func NewRank[K comparable, V SortableInt]() *RankBase[K, V] {
	return &RankBase[K, V]{
		rankMain: skiplist.NewSkipList[K](),
		dict:     make(map[K]V),
	}
}

func (rb *RankBase[K, V]) AddRanker(e *Ranker[K, V]) (err error) {
	e.rankPtr = rb
	defer func() {
		if err == nil {
			rb.dict[e.Key()] = e.Value
		}
	}()
	return rb.rankMain.Add(e)
}

func (rb *RankBase[K, V]) RemoveRanker(e *Ranker[K, V]) (err error) {
	defer func() {
		if err == nil {
			delete(rb.dict, e.Key())
		}
	}()
	return rb.rankMain.DeleteByKey(e.Key())
}

func (rb *RankBase[K, V]) RemoveRankerByKey(k K) (err error) {
	defer func() {
		if err != nil {
			delete(rb.dict, k)
		}
	}()
	return rb.rankMain.DeleteByKey(k)
}

func (rb *RankBase[K, V]) UpdateRankerData(newData IRanker[K]) (err error) {
	err = rb.rankMain.DeleteByKey(newData.Key())
	if err != nil {
		return
	}
	newData.rankPtr = rb
	defer func() {
		if err == nil {
			rb.dict[e.Key()] = e.Value
		}
	}()
	return rb.rankMain.Add(newData)
}

func (rb *RankBase[K, V]) GetRank(rankerKey K) (ret int32, err error) {
	return rb.rankMain.GetRankByKey(rankerKey)
}

func (rb *RankBase[K, V]) GetReverseRank(rankerKey K) (ret int32, err error) {
	return rb.rankMain.GetReverseRankByKey(rankerKey)
}

func (rb *RankBase[K, V]) Range(startAt int32, endAt int32) (ret []IRanker[K], err error) {
	nds, err := rb.rankMain.GetRange(startAt, endAt)
	if err != nil {
		return
	}
	for _, nd := range nds {
		ndv, ok := nd.(IRanker[K])
		if !ok {
			panic("RankBase::Range error: existing element from GetRange is not kind of RankerBase")
		}
		ret = append(ret, ndv)
	}
	return
}

func (rb *RankBase[K, V]) GetAllRankers() (ret []*Ranker[K, V], err error) {
	return rb.Range(1, rb.rankMain.GetElementsCount())
}

func (rb *RankBase[K, V]) GetRankerDataByRank(rank int32) (ret *Ranker[K, V], err error) {
	nd, err := rb.rankMain.GetElementByRank(rank)
	if err != nil {
		return
	}
	ndv, ok := nd.(*Ranker[K, V])
	if !ok {
		panic("RankBase::Range error: existing element from GetRange is not kind of RankerBase")
	}
	return ndv, nil
}

func (rb *RankBase[K, V]) GetRankerDataByKey(rankerKey K) (ret *Ranker[K, V], err error) {
	nd, err := rb.rankMain.GetElementByKey(rankerKey)
	if err != nil {
		return
	}
	ndv, ok := nd.(*Ranker[K, V])
	if !ok {
		panic("RankBase::Range error: existing element from GetElementByKey is not kind of RankerBase")
	}
	return ndv, nil
}
func (rb *RankBase[K, V]) Print() {
	rb.rankMain.Print()
}
