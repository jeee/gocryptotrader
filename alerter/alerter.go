package alerter 

import (
    "bytes"
    "fmt"
    "os/exec"
    "text/template"
    "sync"
    "strings"
    "time"
     
    //"github.com/thrasher-/gocryptotrader/currency/pair"
    //ticker "github.com/thrasher-/gocryptotrader/exchanges/ticker"
    exchange "github.com/thrasher-/gocryptotrader/exchanges"
    "github.com/thrasher-/gocryptotrader/config"
) 

const (
    LESS = "LESS"       //  <
    GREATER = "GREATER" //  >
)

var alerts []config.Alert

var botconfig *config.Config

// type ExchangeTicker struct {
//   Exchange string
//   Ticker   ticker.Price
// }

type singletonTickerPriceChannel struct {
	Channel chan interface{}
	mu    sync.RWMutex
}

var (
    once sync.Once
    r *singletonTickerPriceChannel
)

func GetChannel() *singletonTickerPriceChannel {
	once.Do(func() {
		r = &singletonTickerPriceChannel{
          Channel: make(chan interface{}, 1),
        }
	})

	return r
}


func AlerterRoutine(config  *config.Config) {
  tickerPriceChannel:= GetChannel().Channel
  botconfig = config
  timer := time.NewTicker(5 * time.Second)
  
  quit := make(chan struct{})
  go func() {
      for {
        select {
          case <- timer.C:
              // play alerts if they are set to RUNNING
              alerts = config.Alerter.Alerts
              for i:=0; i < len(alerts); i++{
                alert := alerts[i]
                if strings.ToUpper(alert.State) == "RUNNING" {
                  repeatAlert(alert)
                }
              }
          case <- quit:
              timer.Stop()
              return
          }
      }
  }()  
  
  for {
    exchangeprice := <- tickerPriceChannel
    fmt.Println("Alerter received price: ", exchangeprice)
    
    ep, ok := exchangeprice.(exchange.TickerData)
    if ok {
        alertCheck(ep)
    }
  }
}


// //pairMap := alerter.CreatePairMap(b.EnabledPairs)
// //Pair         : pairMap[strings.ToLower(tickerstr.Symbol)],
// func CreatePairMap(enabledPairs []string) map[string]pair.CurrencyPair{
//   pairMap := make(map[string]pair.CurrencyPair)
//   for i:=0;i<len(enabledPairs);i++{
//     cp := pair.NewCurrencyPairFromString(enabledPairs[i])
//     pairMap[cp.Display("", false).String()] = cp
//   }
//   return pairMap
// }
// 
// func CreatePairMapWithMinus(enabledPairs []string) map[string]pair.CurrencyPair{
//     pairMap := make(map[string]pair.CurrencyPair)
//     for i:=0;i<len(enabledPairs);i++{
//       cp := pair.NewCurrencyPairFromString(enabledPairs[i])
//       //log.Println(
//       c := strings.ToLower(enabledPairs[i])
//       pairMap[c[0:3] + "-" + c[3:]] = cp
//     }
//   return pairMap
// }
// 
func ReplaceDelimiters(pair string) string{
  pair = strings.Replace(pair, "-", "", -1)
  pair = strings.Replace(pair, "-", "", -1)
  return pair
}

func repeatAlert(alert config.Alert){
  type Parameters struct {
	Alert config.Alert
	Symbol string
  }
  
  parameters := Parameters{alert, ReplaceDelimiters(alert.Pair)}
  tmpl, err := template.New("test").Parse(botconfig.Alerter.TemplateRepeat)
  if err != nil { panic(err) }
  
  var tpl bytes.Buffer
  
  if err := tmpl.Execute(&tpl, parameters); err != nil {
      panic( err)
  }

  command := tpl.String()

  _, err = exec.Command("sh","-c",command).Output()
  fmt.Println("running command: ", command)
    if err != nil {
        fmt.Println("error occured")
        fmt.Printf("%s", err)
    }
}

func raiseAlert(alert *config.Alert, price exchange.TickerData){
     fmt.Println(">>>>>: ", price)
  alert.SetRunning()
  type Parameters struct {
	Ticker exchange.TickerData
	Alert config.Alert
	Symbol string
  }
  
  parameters := Parameters{price, *alert, ReplaceDelimiters(alert.Pair)}
  tmpl, err := template.New("test").Parse(botconfig.Alerter.Template)
  if err != nil {
      fmt.Errorf("alerter.go - Unable to load template from config. Error: %s", err) }
  
  var tpl bytes.Buffer
  
  if err := tmpl.Execute(&tpl, parameters); err != nil {
      fmt.Errorf("alerter.go - Unable to execute template from config. Error: %s", err)
  }

  command := tpl.String()
fmt.Println(">>>>>---: ", price)
  _, err = exec.Command("sh","-c",command).Output()
  fmt.Println("running command: ", command)
    if err != nil {
        fmt.Println("error occured")
        fmt.Printf("%s", err)
    }
}

func alertCheck(tickerData exchange.TickerData) {
  for i:=0; i<len(alerts);i++ {
    alert := &alerts[i]
    if alert.Exchange == tickerData.Exchange {
      if ReplaceDelimiters(tickerData.Pair.String()) == alert.Pair {
        if alert.State == "active" {
          if alert.Type == LESS && alert.Value >= tickerData.ClosePrice {
            go raiseAlert(alert, tickerData)
          } else if alert.Type == GREATER && alert.Value <= tickerData.ClosePrice {
            go raiseAlert(alert, tickerData)
          }
        }
      }
    }
  }
}
