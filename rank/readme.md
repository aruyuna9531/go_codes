模板化的排行榜

如果上榜人还需要记录别的数据，在上层结构体里定义Ranker指针就行了
```go
type MyRanker struct {
	*RankerBase[int, int]
	// other ranker data
}
```