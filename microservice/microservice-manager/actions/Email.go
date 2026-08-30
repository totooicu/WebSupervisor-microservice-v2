package actions
import(
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/go-mytool/config"
	"fmt"
	"log"
)



// type EmailConfig struct {
//     Host               string `json:"host"`
//     Port               int    `json:"port"`
//     Username           string `json:"username"`
//     Password           string `json:"password"`
//     WaitTimeMS         int    `json:"wait_time_ms"`
// }

// type EmailContent struct{
// 	Tos       []string `json:"tos"`
//     Subject  string `json:"subject"`
//     Body     string `json:"body"`
// }
// type EmailRequestByConfig struct {//根据配置发送邮件
// 	EmailChoose string `json:"email_choose"`//空为default_email配置
// 	EmailContent EmailContent `json:"email_content"`
// }
// type EmailRequestByCustom struct {//根据自定义配置发送邮件
// 	EmailConfig EmailConfig `json:"email_config"`
//     EmailContent EmailContent `json:"email_content"`
// }



var content=map[string]any{
		"tos":[]any{"21@qq.com","202cn"},
		"subject":"hello",
		"body":"hello world",
	}
func sendEmailByConfig() {
	choose:=""
	resp,e:=stream.Send("email-stream","email_by_config",map[string]interface{}{
		"email_content":content,
		"email_choose":choose,
	},0)
	if e!= nil{
		log.Fatal("send email by config error:", e)
	}
	fmt.Println(resp)
}
func sendEmailByCustom(){
	
	var econfig=map[string]any{
		"host":"smtp.qq.com",
		"port":465,
		"username":config.ExpandEnvVars("${QQ_MAIL_ACCOUNT_1134}"),
		"password":config.ExpandEnvVars("${QQ_MAIL_PASSWORD_1134}"),
		"wait_time_ms":10000,
	}
	resp,e:=stream.Send("email-stream","email_by_custom",map[string]interface{}{
		"email_config":econfig,
		"email_content":content,
	},0)
	if e!= nil{
		log.Fatal("send email by custom error:", e)
	}
	fmt.Println(resp)
}