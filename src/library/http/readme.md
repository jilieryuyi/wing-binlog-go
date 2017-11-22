登录测试，生成session，cookie为user_sign
http://localhost:9989/user/login?username=admin&password=admin
退出登录，删除cookie user_sign，删除session
http://localhost:9989/user/logout