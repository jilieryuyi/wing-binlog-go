CREATE TABLE `cursor` (
 `bin_file` varchar(128) NOT NULL DEFAULT '' COMMENT '最后读取到的binlog文件名',
 `pos` bigint(20) NOT NULL DEFAULT '0' COMMENT '读取到的位置',
 `event_index` bigint(20) NOT NULL DEFAULT '0' COMMENT '最新的事件索引',
 `updated` int(11) NOT NULL DEFAULT '0' COMMENT '最后更新的时间戳'
) ENGINE=InnoDB DEFAULT CHARSET=utf8