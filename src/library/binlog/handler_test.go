package binlog

import "testing"

func TestGetPoint(t *testing.T) {
	//h := &binlogHandler{}
	//p , err := h.getPoint("double", 1.1)
	//
	//if err != nil {
	//	t.Errorf("发生错误：%+v", err)
	//}
	//
	//if p != 1 {
	//	t.Errorf("获取小数位长度不对-1")
	//}
	//
	//p , err = h.getPoint("double", 1.12)
	//
	//if err != nil {
	//	t.Errorf("发生错误：%+v", err)
	//}
	//
	//if p != 1 {
	//	t.Errorf("获取小数位长度不对-2")
	//}
	//
	//p , err = h.getPoint("decimal(10,5)", 0)
	//
	//if err != nil {
	//	t.Errorf("发生错误：%+v", err)
	//}
	//
	//if p != 5 {
	//	t.Errorf("获取小数位长度不对-2")
	//}
	//
	//
	//p , err = h.getPoint("float(10,5)", 0)
	//
	//if err != nil {
	//	t.Errorf("发生错误：%+v", err)
	//}
	//
	//if p != 5 {
	//	t.Errorf("获取小数位长度不对-2")
	//}
	//

}

func TestBinlogHandler_SaveBinlogPostionCache(t *testing.T) {
	binfile := "mysql-bin.000059"
	pos := int64(123456)
	eventIndex := int64(20)

	h := &binlogHandler{}
	h.SaveBinlogPostionCache(binfile, pos, eventIndex)

	s, p, e := h.getBinlogPositionCache()

	if s != binfile || pos != p || e != eventIndex {
		t.Errorf("getBinlogPositionCache error")
	}
}
