package types

// NewUserFileListRequest 构建用户文件列表请求。
func NewUserFileListRequest(id int64, page, size int) UserFileListRequest {
	return UserFileListRequest{
		Id:   id,
		Page: page,
		Size: size,
	}
}

// NewLoginRequest 构建登录请求。
func NewLoginRequest(name, password string) LoginRequest {
	return LoginRequest{
		Name:     name,
		Password: password,
	}
}
