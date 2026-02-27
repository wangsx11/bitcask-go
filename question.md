db.Get()部分
根据key找value，如果一个key被存入多次，怎么判断获取到的是最新的数据？


学习到 第十三节 1小时处


iterator := db.index.Iterator(false)
defer iterator.Close() 

每一个初始化iterator的部分，都需要加上defer iterator.Close()， 这部分不是很明白

// 在Put和Delete方法中，对于传入的Key，都修改为了 
    Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo)
    那么，在Get方法中，是不是也需要修改？
