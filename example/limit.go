package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

// SCRIPT argv 1:rate
// SCRIPT argv 2:burst
const SCRIPT = `
function string:nsplit(sep)
    local rt={}
    local pattern = string.format("([^%s]+)", sep)
    self:gsub(pattern, function(w) table.insert(rt, tonumber(w)) end )
    return rt
end

local timestamp = redis.call("TIME")
local now=tonumber(timestamp[1])*1000000+tonumber(timestamp[2])

local rate=ARGV[1]
local burst=ARGV[2]*1000000 
local val = redis.call("get", KEYS[1])
local dict = {}

local delta=0
if val == false or val==nil then
	dict[1]=-1000000
else
	dict = string.nsplit(val, '|')
	delta=now-dict[2]
end

local exc=dict[1]
exc=exc-rate*delta+1000000
if exc<0 then
	exc=0
end
local resp=0
if exc>burst then 
	resp=1
else
	dict[1]=exc
	dict[2]=now
	local str=""
	for k, v in pairs(dict) do
		str=str  .. v ..'|'
	end
	redis.call("setex", KEYS[1],  math.ceil(ARGV[2]/ARGV[1])+2, str)
end
return resp
`

func main() {
	rate := 5000.0
	burst := 1000.0
	loop := 40000
	key := "test_key"
	
	
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:8000",
		Password: "RxRx9zz",
		DB:       1,
	})

	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(SCRIPT)))
	fmt.Printf("scriptSHA1:%s\n", scriptSHA1)
	if !client.ScriptExists(scriptSHA1).Val()[0] {
		var err error
		scriptSHA1, err = client.ScriptLoad(SCRIPT).Result()
		fmt.Printf("load scriptSHA1:%v, err:%v\n", scriptSHA1, err)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Printf("find scriptSHA1:%s\n", scriptSHA1)
	}
	client.Del(key)
	
	start := time.Now()
	allow := 0
	
	for i := 0; i < loop; i++ {
		r2, _ := client.EvalSha(scriptSHA1, []string{key}, rate, burst).Result()
		v2, flag := r2.(int64)
		if !flag {
			fmt.Printf("undef:%T", r2)
			panic("undef resp")
		}
		if v2 == 0 {
			allow++
		}	
		if i != loop-1 && i%10==0 && loop<1000 * 50 {
			time.Sleep(time.Microsecond*time.Duration(1))
		}
	}
	els:=time.Since(start)
	fmt.Println("run",els)
	expect := rate*float64(time.Since(start)/time.Millisecond)/1000 + burst
	fmt.Printf("loop:%d, allow:%d, expect:%0.1f\n", loop, allow, expect)
	fmt.Printf("error:%0.1f(%%), times:%0.1f\n",100-100*expect/float64(allow),expect-float64(allow))
	if els<time.Second {
		fmt.Printf("evaluation requires more cycles\n")
	}
}
