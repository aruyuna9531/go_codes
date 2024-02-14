有序map的实现

底层用跳表结构（类似和redis的大型zset）←大型：指zset整体size超过zset-max-ziplist-entries定义值或zset里至少一个key值的size超过zset-max-ziplist-value定义值。redis会在一个zset满足上述条件将底层从ziplist改成skiplist，并且后续操作使该zset不满足上述条件时，底层结构不会逆转回ziplist。
redis用跳表作为zset的数据结构的原因是会有取分数排名区间的一套数据的情况（zrange/zrevrange）。redis原作者的话：
```
There are a few reasons:
1) They are not very memory intensive. It's up to you basically. Changing parameters about the probability of a node to have a given number of levels will make then less memory intensive than btrees.
2) A sorted set is often target of many ZRANGE or ZREVRANGE operations, that is, traversing the skip list as a linked list. With this operation the cache locality of skip lists is at least as good as with other kind of balanced trees.
3) They are simpler to implement, debug, and so forth. For instance thanks to the skip list simplicity I received a patch (already in Redis master) with augmented skip lists implementing ZRANK in O(log(N)). It required little changes to the code.
```

redis在大型zset上用的结构是dict+skiplist
dict是辅助查分的（根据key查score，如zscore）使这些针对单个数据的操作依然获得O(1)的复杂度