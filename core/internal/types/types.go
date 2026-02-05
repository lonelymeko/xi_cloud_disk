// goctl 生成代码，请勿手动修改。
// goctl 1.9.2

package types

// CreateShareRecordRequest 创建分享记录请求。
type CreateShareRecordRequest struct {
	Identity    string `json:"identity"`
	ExpiredTime int    `json:"expired_time"`
}

// CreateShareRecordResponse 创建分享记录响应。
type CreateShareRecordResponse struct {
	Identity string `json:"identity"`
}

// GetShareRecordRequest 获取分享记录请求。
type GetShareRecordRequest struct {
	Identity string `json:"identity"`
}

// GetShareRecordResponse 获取分享记录响应。
type GetShareRecordResponse struct {
	RepositoryIdentity string `json:"repository_identity"`
	Name               string `json:"name"`
	Ext                string `json:"ext"`
	Size               int64  `json:"size"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Name     string `json:"name"`     // 用户名或邮箱
	Password string `json:"password"` // 密码
}

// LoginResponse 登录响应。
type LoginResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Name     string `json:"name"`     // 用户名
	Email    string `json:"email"`    // 邮箱
	Password string `json:"password"` // 密码
	Code     string `json:"code"`
}

// RegisterResponse 注册响应。
type RegisterResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// ChangePasswordRequest 修改密码请求。
type ChangePasswordRequest struct {
	Identity    string `json:"identity"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordResponse 修改密码响应。
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// ResetPasswordRequest 重置密码请求。
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// ResetPasswordResponse 重置密码响应。
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// SaveResourceRequest 保存资源请求。
type SaveResourceRequest struct {
	RepositoryIdentity string `json:"repository_identity"`
	ParentId           int64  `json:"parent_id"`
	Name               string `json:"name"`
}

// SaveResourceResponse 保存资源响应。
type SaveResourceResponse struct {
	Identity string `json:"identity"`
}

// SendVerificationCodeRequest 发送验证码请求。
type SendVerificationCodeRequest struct {
	Email string `json:"email"` // 邮箱地址
}

// SendVerificationCodeResponse 发送验证码响应。
type SendVerificationCodeResponse struct {
	Message string `json:"message"` // 返回消息
}

// UploadFileRequest 上传文件请求。
type UploadFileRequest struct {
	Hash      string `json:"hash,optional" form:"hash,optional"`
	Name      string `json:"name,optional" form:"name,optional"`
	Ext       string `json:"ext,optional" form:"ext,optional"`
	Size      int64  `json:"size,optional" form:"size,optional"`
	ObjectKey string `json:"object_key,optional" form:"object_key,optional"`
	ParentId  int64  `json:"parent_id,optional" form:"parent_id,optional"`
}

// UploadFileResponse 上传文件响应。
type UploadFileResponse struct {
	Message string `json:"message,omitempty"`
}

// UserDetailRequest 用户详情请求。
type UserDetailRequest struct {
	Identity string `json:"identity"`
}

// UserDetailResponse 用户详情响应。
type UserDetailResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserFile 用户文件信息。
type UserFile struct {
	Id                 int64  `json:"id"`
	Identity           string `json:"identity"`
	Name               string `json:"name"`
	Ext                string `json:"ext"`
	Size               int64  `json:"size"`
	RepositoryIdentity string `json:"repository_identity"`
}

// UserFileListRequest 用户文件列表请求。
type UserFileListRequest struct {
	Id   int64 `json:"id"`
	Page int   `json:"page"`
	Size int   `json:"size"`
}

// UserFileListResponse 用户文件列表响应。
type UserFileListResponse struct {
	List  []*UserFile `json:"list"`
	Count int64       `json:"count"`
}

// UserFileMoveRequest 用户文件移动请求。
type UserFileMoveRequest struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
	ParentId int64  `json:"parent_id"`
}

// UserFileMoveResponse 用户文件移动响应。
type UserFileMoveResponse struct {
}

// UserFileNameUpdateRequest 用户文件名更新请求。
type UserFileNameUpdateRequest struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
}

// UserFileNameUpdateResponse 用户文件名更新响应。
type UserFileNameUpdateResponse struct {
}

// UserFolderCreateRequest 用户文件夹创建请求。
type UserFolderCreateRequest struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name"`
}

// UserFolderCreateResponse 用户文件夹创建响应。
type UserFolderCreateResponse struct {
	Id       int64  `json:"id"`
	Identity string `json:"identity"`
}

// UserFolderDeleteRequest 用户文件夹删除请求。
type UserFolderDeleteRequest struct {
	Identity string `json:"identity"`
}

// UserFolderDeleteResponse 用户文件夹删除响应。
type UserFolderDeleteResponse struct {
}

// DownloadURLRequest 下载链接请求。
type DownloadURLRequest struct {
	RepositoryIdentity string `json:"repository_identity"`
	Expires            int    `json:"expires"`
}

// DownloadURLResponse 下载链接响应。
type DownloadURLResponse struct {
	URL     string `json:"url"`
	Expires int    `json:"expires"`
}

// ShareDownloadURLRequest 分享下载链接请求。
type ShareDownloadURLRequest struct {
	ShareIdentity string `json:"share_identity"`
	Expires       int    `json:"expires"`
}

// ShareDownloadURLResponse 分享下载链接响应。
type ShareDownloadURLResponse struct {
	URL     string `json:"url"`
	Expires int    `json:"expires"`
}
