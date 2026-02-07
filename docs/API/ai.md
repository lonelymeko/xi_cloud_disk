# API-AI
BASE_USERS http://127.0.0.1:8888/api/users
BASE_FILE  http://127.0.0.1:8888/api/file
BASE_SHARE http://127.0.0.1:8888/api/share
AUTH Authorization: Bearer <token>
RESP {code,msg,data}

USERS
POST /login            body{name,password} -> data{token,name}
POST /register         body{name,email,password,code} -> data{token,name}
POST /send-verification-code body{email} -> data{message}
POST /password/reset   body{email,code,new_password} -> data{message}
POST /detail           body{identity} -> data{name,email}
POST /password/update  [auth] body{identity,old_password,new_password} -> data{message}

FILE
POST /upload           [auth] multipart file -> data{message} (async enqueue, limit 10GB)
POST /url              [auth] body{repository_identity,expires?} -> data{url,expires} (expires<=0=>3600,max=604800)
POST /user/list        [auth] body{id,page,size} -> data{list:UserFile[],count}
PUT  /user/file/move   [auth] body{identity,name,parent_id} -> data{}
POST /user/file/name/update [auth] body{identity,name} -> data{}
POST /user/folder/create [auth] body{parent_id,name} -> data{id,identity}
DELETE /user/folder/delete [auth] body{identity} -> data{}
UserFile{ id,identity,name,ext,size,repository_identity }

SHARE
POST /create [auth] body{identity(repository_identity),expired_time} -> data{identity}
GET  /get?identity=... public -> data{repository_identity,name,ext,size}
POST /url  public body{share_identity,expires?} -> data{url,expires} (expires<=0=>3600,max=604800)
POST /save [auth] body{repository_identity,parent_id,name} -> data{identity}
