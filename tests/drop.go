package main

import (
	//"strings"
	//"fmt"
	"regexp"
	"log"
	"bytes"
)

func main() {
	//expDropTable   = regexp.MustCompile("(?i)^DROP(\\s){1,}TABLE((\\s){1,}IF(\\s){1,}EXISTS){0,1}(\\s){1,}`{0,1}(.*?)`{0,1}\\.{0,1}`{0,1}([^`\\.]+?)`{0,1}($|\\s)")

	//expDropTable  := regexp.MustCompile("(?i)^DROP\\sTABLE(\\s{1,}IF\\s{1,}EXISTS){0,1}\\s.*?`{0,1}(.*?)`{0,1}\\.{0,1}`{0,1}([^`\\.]+?)`{0,1}\\s.*")
	expDropTable   := regexp.MustCompile("(?i)^DROP\\sTABLE(\\sIF\\sEXISTS){0,1}\\s`{0,1}(.*?)`{0,1}\\.{0,1}`{0,1}([^`\\.]+?)`{0,1}($|\\s)")
	cases := []string{
		"drop table test1",
		"DROP TABLE test1",
		"DROP TABLE test1",
		"DROP table IF EXISTS test.test1",
		"drop table `test1`",
		"DROP TABLE `test1`",
		"DROP table IF EXISTS `test`.`test1`",
		"DROP TABLE `test1` /* generated by server */",
		"DROP table if exists test1",
		"DROP table if exists `test1`",
		"DROP table if exists test.test1",
		"DROP table if exists `test`.test1",
		"DROP table if exists `test`.`test1`",
		"DROP table if exists test.`test1`",
		"DROP table if exists test.`test1`",
	}

	table := []byte("test1")
	for _, s := range cases {
		m := expDropTable.FindSubmatch([]byte(s))
		//log.Println(m)
		//for _, v :=range m{
		//	fmt.Println(string(v))
		//}
		//fmt.Println(len(m),"=================")

		if m == nil {
			log.Fatalf("TestDropTableExp: case %s failed\n", s)
			return
		}
		if len(m) < 4 {
			log.Fatalf("TestDropTableExp: case %s failed\n", s)
			return
		}
		if !bytes.Equal(m[3], table) {
			log.Fatalf("TestDropTableExp: case %s failed\n", s)
		}
	}

}
