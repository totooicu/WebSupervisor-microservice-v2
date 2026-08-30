
package actions
import(
	"log"

	"github.com/totooicu/go-mytool/stream"
)
func ping(){
	r,e:=stream.Send("parser-stream","ping",map[string]any{},0)
	if e!=nil{log.Println(e)}
	log.Println(r.Playload)
}