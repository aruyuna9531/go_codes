package rank

import (
	"log"
	"testing"
	"time"
)

func TestRank(t *testing.T) {
	r := NewRank[int, int]()
	r.AddRanker(&Ranker[int, int]{
		RankerId:   1,
		Value:      2,
		UpdateTime: time.Now().UnixMilli(),
	})
	r.AddRanker(&Ranker[int, int]{
		RankerId:   2,
		Value:      36,
		UpdateTime: time.Now().UnixMilli(),
	})
	r.AddRanker(&Ranker[int, int]{
		RankerId:   3,
		Value:      19,
		UpdateTime: time.Now().UnixMilli(),
	})
	r.AddRanker(&Ranker[int, int]{
		RankerId:   4,
		Value:      25,
		UpdateTime: time.Now().UnixMilli(),
	})
	r.Print()
	gr, _ := r.GetRank(3)
	log.Println(gr) // 3
	xs, _ := r.Range(2, 4)
	for _, x := range xs {
		xr, _ := x.GetRank()
		log.Printf("[%d,%d,%d] ", x.RankerId, x.Value, xr) //4/25,3/19,1/2
	}
	log.Printf("\n")
	r.UpdateRankerData(&Ranker[int, int]{
		RankerId:   3,
		Value:      38,
		UpdateTime: time.Now().UnixMilli(),
	})
	gr, _ = r.GetRank(3)
	log.Println(gr) // 1
	xs, _ = r.Range(2, 4)
	for _, x := range xs {
		xr, _ := x.GetRank()
		log.Printf("[%d,%d,%d] ", x.RankerId, x.Value, xr) //2/36,4/25,1/2
	}
	log.Printf("\n")
}
