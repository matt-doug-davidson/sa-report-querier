# SA Report Querier


```bash
CONFIG=config.yaml go run sa-report-querier.go
```

```bash
go build sa-report-querier.go
CONFIG=config.yaml ./sa-report-querier
```

## Configuration

### Sections
|Section |  Required  |  Description  |
|:--:|:--:|:--:|
| sa | yes | Section Used to configure the interfaces into Swagger and MQTT |
| report | yes | Section used to define the transit reports |
| debug | yes | Enable debug. This is a boolean.

#### SA
|Key |  Required  |  Description  |
|:--:|:--:|:--:|
|host |yes | The host for Swagger and MQTT. IT is used for both if mqtt-host is not set |
|swagger-port| yes | This is the TCP port used to access Swagger |
| swagger-username | yes | The username used to access the database using Swagger|
| swagger-password | yes | The password used to access the database using Swagger |
| mqtt-port | yes | The port used to sent MQTT messages to the database |
| mqtt-client-id | yes| The unique identifier used by this MQTT client |
| mqtt-host | No | The host that MQTT messages will be sent to. If not defined, the host is used |


#### Reports

|Key |  Required  |  Description  |
|:--:|:--:|:--:|
| endpoints | yes | This is a list of ingress and egress entity path pairs. |
| ingress-chunk-interval| yes | The interval that a chunk of data will be pulled from the database |
| ingress-chunks| Yes | The number of chunks used to check for transits. If interval is 5 and the chunks are 4 then 20 minutes of data will be evaluated. However, it could be as low as 15 as the vehicle could ingress near the end of the first interval and could egress near the beginning of the last held interval. |

##### Endpoints

|Key |  Required  |  Description  |
|:--:|:--:|:--:|
| ingress | yes | The starting point of the transit observation |
| egress | yes | Then ending point of the transit observation |


```yaml
---
  sa:
     host: "10.15.0.225"
     swagger-port: 10000
     swagger-username: googlie
     swagger-password: "StartIt$"
     mqtt-port: 1883
     mqtt-client-id: "sa_Report_querier"
  report:
    endpoints:
      -
        ingress: /Bucharest/Lujerului/CVL_LujeruluiOuter
        egress:  /Bucharest/Praktiker/CVL_PraktikerOuter
      -
        ingress: /Bucharest/Praktiker/CVL_PraktikerInner
        egress: /Bucharest/Lujerului/CVL_LujeruluiInner
    ingress-chunk-interval: 5
    ingress-chunks: 4
```
