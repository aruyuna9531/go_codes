排行榜基本结构

如果上榜人还需要记录别的数据，在上层结构体里定义Ranker指针就行了，RankerBase已经实现ISkipListElement接口，可以直接用作skiplist的元素
```go
type MyRanker struct {
	*RankerBase[int, int]
	// other ranker data
}
```

基本使用 QuickStart

（用之前先在项目里go get github.com/aruyuna9531/skiplist）

```go
r := NewRank[int, int]()

// 添加一个排行实体
r.AddRanker(&Ranker[int, int]{
    RankerId:   1,
    Value:      2,
    UpdateTime: time.Now().UnixMilli(),
})
// 添加其他的排行实体...

// 获得某个ranker在榜上的排名（不存在时返回err）
gr, err := r.GetRank(1)

// 获得某个ranker在榜上的倒序排名
gr, err := r.GetReverseRank(1)

// 获得某个排名的ranker数据
data, err := r.GetRankerDataByRank(1)

// 更新榜上ranker数据
r.UpdateRankerData(&Ranker[int, int]{
RankerId:   3,
Value:      38,
UpdateTime: time.Now().UnixMilli(),
})

// 从榜上删除ranker
r.RemoveRankerByKey(1)
```