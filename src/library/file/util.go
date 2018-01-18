package file

import "os"

// check file is exists
// if exists return true, else return false
func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return false
		} else {
			// other error
			return false
		}
	} else {
		//exist
		return true
	}
}
