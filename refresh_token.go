package main

import (
	"email-forwarding/services"
	"fmt"
	"log"
)

func main() {
	fmt.Println("=== Gmail Token 刷新工具 ===")
	
	// 设置代理
	services.SetProxy("http://127.0.0.1:10810")
	
	fmt.Println("正在刷新Gmail OAuth2 token...")
	fmt.Println("请确保您有有效的 credentials.json 文件")
	
	// 尝试创建Gmail服务，这会自动刷新token
	_, err := services.NewGmailService("credentials.json", "token.json", "your-email@gmail.com")
	if err != nil {
		log.Printf("Token刷新失败: %v", err)
		fmt.Println("\n可能的原因:")
		fmt.Println("1. credentials.json 文件不存在或无效")
		fmt.Println("2. 代理配置不正确")
		fmt.Println("3. 网络连接问题")
		fmt.Println("4. Google OAuth2 配置问题")
		return
	}
	
	fmt.Println("Token刷新成功！")
	fmt.Println("现在可以正常运行主程序了。")
} 