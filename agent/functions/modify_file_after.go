package functions

import "github.com/clover0/issue-agent/store"

func StoreFileAfterModifyFile(s *store.Store, file store.File) {
	StoreFileAfterPutFile(s, file)
}
