﻿package thunder

import (
	"fmt"
	"github.com/ssp97/Ka-ineshizuku-Project/pkg/zero"
	ZeroBot "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type thunder struct {
	onlineList  []int64
	question	string
	answer		string
	victim		int64
	lastVictim  int64
	referee		int64
	nowTime		time.Time
}
var thunderList = map[int64]thunder{}
var lock sync.Mutex
type Config struct {
	Enable bool
}

func Init(c Config) {

	if c.Enable == false {
		return
	}

	zero.Default().OnRegex("^手捧雷$", ZeroBot.OnlyGroup).SetPriority(1).Handle(func(ctx *ZeroBot.Ctx) {

		if zero.IsGroupManager(ctx) == false{
			ctx.SendChain(message.Text("服务受限：非群管理员"))
			return
		}

		group := ctx.Event.GroupID
		gameTime := 60 + rand.Intn(160)
		level := 3

		lock.Lock()
		_,ok := thunderList[group]
		q,a := questionMake(level)
		t := thunder{
			question: q,
			answer: a,
			victim: ctx.Event.UserID,
			nowTime: time.Now(),
			referee: ctx.Event.SelfID,
			//onlineList: onlineList,
		}
		if ok == false{
			thunderList[group] = t
		}
		lock.Unlock()

		//fmt.Println(t)
		if ok==true{
			ctx.SendChain(message.Text("场上已经有雷了"))
			return
		}

		ctx.SendChain(message.Text(fmt.Sprintf("手捧雷游戏现在开始，游戏一共%d秒，回答正确，即可将雷传到其他人手中，准备好了吗？游戏即将开始,预备！",gameTime)))
		time.Sleep(5 * time.Second)

		ctx.SendChain(message.At(t.victim),message.Text(t.question))

		go func(ctx *ZeroBot.Ctx, group int64) {

			time.Sleep(1*time.Second)

			t := thunderList[group]
			startTime := time.Now().Unix()
			stopTime := startTime + int64(gameTime)
			for {
				next := ZeroBot.NewFutureEvent("message", -9999, true, ZeroBot.CheckUser(t.victim), func(ctx *ZeroBot.Ctx) bool {
					if ctx.Event.GroupID == group && ctx.Event.SelfID == t.referee{
						return true
					}
					return false
				})

				recv, cancel := next.Repeat()
				WaitAnswer:
				for {
					select {
					case <- time.After(time.Second * time.Duration(stopTime - time.Now().Unix())):
						ctx.SendChain(message.Text("手捧雷BOOM，"),
							message.At(t.victim),
							message.Text(fmt.Sprintf("菊花残，满地伤，躺下%d秒捂菊花",gameTime)))
						ctx.SetGroupBan(
							group,
							t.victim, // 要禁言的人的qq
							int64(gameTime),
						)
						cancel()
						lock.Lock()
						delete(thunderList, group)
						lock.Unlock()
						return
					case e := <-recv:
						//cancel()
						newCtx := &ZeroBot.Ctx{Event: e, State: ZeroBot.State{}}
						reg := regexp.MustCompile(t.answer)
						if reg.Match([]byte(newCtx.Event.Message.String())){
							ctx.SendChain(message.At(t.victim),
							message.Text(fmt.Sprintf("(%d)回答正确，来。你要把雷丢给谁？",t.nowTime.Unix())))
							break WaitAnswer
						}else{
							ctx.SendChain(
								//message.At(t.victim),
								message.Text(fmt.Sprintf("回答错误，听清楚了，%s",t.question)))
						}
					}
				}
				WaitNextVictim:
				for  {
					select {
					case <- time.After(time.Second * time.Duration(stopTime - time.Now().Unix())):
						ctx.SendChain(message.Text("啊偶，"),
							message.At(t.victim),
							message.Text(fmt.Sprintf("没有及时把雷传出去，手捧雷BOOM，菊花残，满地伤，躺下%d秒捂菊花",gameTime)))
						ctx.SetGroupBan(
							group,
							t.victim, // 要禁言的人的qq
							int64(gameTime),
						)
						cancel()
						lock.Lock()
						delete(thunderList, group)
						lock.Unlock()
						return
					case e := <-recv:
						newCtx := &ZeroBot.Ctx{Event: e, State: ZeroBot.State{}}
						reg := regexp.MustCompile("\\[CQ:at,qq=(\\d+)")
						result := reg.FindAllStringSubmatch(newCtx.Event.Message.String(),-1)
						nextId := int64(0)
						if len(result) > 0{
							nextId = strToInt(result[0][1])
						}
						if newCtx.Event.IsToMe || zero.IsBot(nextId) == true{
							cancel()  // 提前停止监听
							level ++
							gameTime += 90
							q,a := questionMake(level)
							t.question = q
							t.answer = a
							ctx.SendChain(message.At(t.victim),
								message.Text("丢雷失败，并被成功丢了回去"),
								)
							time.Sleep(5*time.Second)
							ctx.SendChain(message.Text("小雫喊出了超级加倍，难度增加了，惩罚时间增加90s"))
							time.Sleep(5*time.Second)
							stopTime += 12
							break WaitNextVictim
						}
						if len(result) <= 0{
							ctx.SendChain(message.At(t.victim),
								message.Text("给谁给谁，我听不清"))
						} else {
							cancel()
							t.lastVictim = t.victim
							t.victim = nextId
							q,a := questionMake(level)
							t.question = q
							t.answer = a
							t.nowTime = time.Now()
							break WaitNextVictim
						}
					}
				}


				if t.victim == 1648468212{ // 小夜不会受伤
					ctx.SendChain(message.Text(fmt.Sprintf("问：%s 答：%s",t.question,t.answer)))
					time.Sleep(5*time.Second)
					ctx.SendChain(message.Text(fmt.Sprintf("问：(%d)%s 答：",t.nowTime.Unix(),"回答正确，来。你要把雷丢给谁？")),
						message.At(t.lastVictim))
					time.Sleep(5*time.Second)
					stopTime += 12
				}

				ctx.SendChain(message.At(t.victim),message.Text(t.question))
			}
		}(ctx, group)

	})
}

func questionMake(level int)(q , a string){
	f := []func(int)(string,string){
		primarySchoolAddition,
		primarySchoolSubtraction,
		primarySchoolMultiplication,
		ChickenAndRabbit,
		JumpInLine,
	}
	q,a = f[rand.Intn(len(f))](level)
	fmt.Printf("%s -> %s\r\n", q, a)
	return
}


func primarySchoolAddition(level int)(q , a string){
	rand.Seed(time.Now().Unix())
	p := math.Pow10(level)
	x := rand.Intn(int(p))
	y := rand.Intn(int(p))
	z := x + y
	if level <= 4{
		q = fmt.Sprintf("小学数学题： %d + %d = ?", x, y)
	}else{
		q = fmt.Sprintf("数学题： %d + %d = ?", x, y)
	}
	a = fmt.Sprintf("%d",z)
	return
}

func primarySchoolSubtraction(level int)(q , a string){
	rand.Seed(time.Now().Unix())
	p := math.Pow10(level-1)
	x := rand.Intn(int(p))
	y := rand.Intn(int(p))
	z := x - y
	if level <= 4{
		q = fmt.Sprintf("小学数学题： %d - %d = ?", x, y)
	}else{
		q = fmt.Sprintf("数学题： %d - %d = ?", x, y)
	}
	a = fmt.Sprintf("%d",z)
	return
}

func primarySchoolMultiplication(level int)(q , a string){
	rand.Seed(time.Now().Unix())
	p := math.Pow10(level/2)
	x := rand.Intn(int(p))
	y := rand.Intn(int(p))
	z := x * y
	if level <= 4{
		q = fmt.Sprintf("小学数学题： %d x %d = ?", x, y)
	}else{
		q = fmt.Sprintf("数学题： %d x %d = ?", x, y)
	}
	a = fmt.Sprintf("%d",z)
	return
}

// 鸡兔同笼
func ChickenAndRabbit(level int)(q , a string){
	rand.Seed(time.Now().Unix())
	p := math.Pow10(level-2) / 2
	chicken := rand.Intn(int(p))
	rabbit := rand.Intn(int(p))
	q = fmt.Sprintf("鸡兔同笼：现有一笼子，里面有鸡和兔子若干只，数一数，共有头%d个，腿%d条，爱看色图的老色pi，你能算出鸡有多少只吗？(回答“x只”)", chicken+rabbit, chicken*2+rabbit*4)
	a = fmt.Sprintf("%d只",chicken)
	return
}

// 插队问题
func JumpInLine(level int)(q , a string){
	rand.Seed(time.Now().Unix())
	p := math.Pow10(level-1)
	m := rand.Intn(int(p)) + 2
	w := rand.Intn(level) + 1
	q = fmt.Sprintf("在一排%d名男同学的队伍中，每两名男同学之间插进%d名女同学，老色pi想一想，可以插多少名女同学？", m, w)
	a = fmt.Sprintf("%d",(m-1)*w)
	return
}

func strToInt(str string) int64 {
	val, _ := strconv.ParseInt(str, 10, 64)
	return val
}