package types

func NewUserFileListRequest(id int64, page, size int) UserFileListRequest {
	return UserFileListRequest{
		Id:   id,
		Page: page,
		Size: size,
	}
}

func NewLoginRequest(name, password string) LoginRequest {
	return LoginRequest{
		Name:     name,
		Password: password,
	}
}
