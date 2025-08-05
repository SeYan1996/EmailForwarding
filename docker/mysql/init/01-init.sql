-- 设置数据库字符集
ALTER DATABASE email_forwarding CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户并授权（如果不存在）
CREATE USER IF NOT EXISTS 'email_user'@'%' IDENTIFIED BY 'email_pass123';
GRANT ALL PRIVILEGES ON email_forwarding.* TO 'email_user'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 显示数据库信息
SHOW DATABASES;
SELECT User, Host FROM mysql.user WHERE User = 'email_user'; 