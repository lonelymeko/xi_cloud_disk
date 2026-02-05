package types

type UploadEvent struct {
	UserIdentity       string `json:"user_identity"`
	ParentId           int64  `json:"parent_id"`
	FilePath           string `json:"file_path"`
	Ext                string `json:"ext"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	IsExisted          bool   `json:"is_existed"`
	RepositoryIdentity string `json:"repository_identity"`
	Hash               string `json:"hash"`
}
