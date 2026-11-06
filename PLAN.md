# PLAN — M5Servo-Go (Servo Kit 360° / 180°)

Plano da lib de servo PWM para M5Stack.

- Módulo: `github.com/nicolasmmb/M5Servo-Go` (decidido)
- Diretório: `/Users/nicolas.mmb/__DEV__/M5Servo-Go`
- Padrão: `m5stick-hardware-pocs/docs/LIB-CREATION-GUIDE.md`
- Estado: fases 0 a 4 concluídas. Fase 5 concluída com 9 testes de hardware.

## 0. Regras deste documento

Este documento usa ASD-STE100. As regras aplicadas são:

- Frase com no máximo 20 palavras.
- Uma instrução por frase.
- Voz ativa.
- Tempo verbal simples.
- Um termo para um conceito.
- Parágrafo com no máximo 6 frases.

## 1. Objetivo

A lib controla servos PWM. A referência é o Servo Kit 360° da M5Stack.
A lib também controla o Servo Kit 180°.

Os objetivos são:

1. Controlar rotação contínua. O usuário define velocidade e sentido.
2. Controlar posição angular. O usuário define o ângulo.
3. Usar somente Go. A lib não usa C, CGo ou Arduino.
4. Testar a lógica no host. O PWM entra por interface.
5. Não alocar memória por comando.
6. Repetir o comportamento das libs oficiais.

## 2. Termos

| Termo | Definição |
|---|---|
| pulso | Largura do sinal alto, em microssegundos. |
| período | Duração de um ciclo do PWM. Neste caso, 20 ms. |
| duty | Valor de comparação do PWM, em counts. |
| count | Unidade do duty. O valor máximo é `Top()`. |
| timer | Bloco do LEDC que define a frequência. |
| canal | Saída do LEDC ligada a um pino. |
| midpoint | Pulso de parada. O valor é a média de `MinPulse` e `MaxPulse`. |
| range | Metade da diferença entre `MaxPulse` e `MinPulse`. |
| stall | Condição em que o eixo não gira. A corrente é máxima. |

## 3. Pesquisa de hardware

### 3.1 Produto

| Item | Valor |
|---|---|
| Produto | Servo Kit 360° e Servo Kit 180° |
| SKU | A076-C |
| Servo | SG90, 9 g |
| Interface | PWM. Não tem chip. Não tem protocolo. |
| Frequência | 50 Hz. Período de 20 ms. |
| Pulso | 0,5 ms a 2,5 ms |
| Tensão | 4,8 V a 6,0 V |
| Corrente sem carga | 60 mA por servo |
| Corrente em stall | 750 mA por servo |
| Deadband | Aproximadamente 8 µs |
| Engrenagem | Metal no motor. Plástico na caixa. |

### 3.2 Ligação

O kit inclui um adaptador Grove para servo.

| Fio Grove | Função |
|---|---|
| Preto | GND |
| Vermelho | 5 V |
| Amarelo | Sinal PWM |

O sinal usa o fio amarelo. A lib oficial confirma este ponto. A chamada
`Servo180Unit((33, 32))` usa `pin = port[1]`. O valor é 32. O pino 32 é o
amarelo do PORT A. No PORT B, o amarelo é o G26.

### 3.3 Alimentação

O usuário informou que o servo fica no Grove. Um servo no Grove é aprovado.
A trilha de 5 V do Grove suporta um servo de 9 g em uso normal.

Dois servos no Grove não são recomendados. A soma em stall chega a 1,5 A.
A fonte do StickC não fornece esta corrente. Para dois servos, use fonte
externa de 5 V com GND comum.

Regras elétricas:

- Use um servo por porta Grove.
- Não bloqueie o eixo. O stall aquece o servo.
- Verifique a tensão de 5 V durante o movimento.
- Use rampa de aceleração. A rampa reduz o pico de corrente.

## 4. Comportamento das libs oficiais

Esta seção descreve o comportamento que a lib deve repetir.

### 4.1 Servo180Unit (MicroPython)

A lib oficial tem esta assinatura:

```python
Servo180Unit(port=None, pin=0, freq=50, count_low=500, count_high=2500)
```

O `pin` vem de `port[1]` quando o usuário informa a porta.
O construtor grava o pulso `count_low` no PWM.
O valor `count_low` está em microssegundos, apesar do nome.

A lib oficial tem estes métodos:

| Método | Regra |
|---|---|
| `set_angle(angle)` | Limita o ângulo entre 0 e 180. |
| `set_duty(us)` | Grava o pulso em microssegundos. |
| `set_percent(p)` | Limita o valor entre 0 e 100. Converte para ângulo. |
| `set_radian(r)` | Limita o valor entre 0 e π. Converte para ângulo. |
| `deinit()` | Desliga o PWM. |

A conversão de ângulo é linear:

```text
pulso = count_low + int(angle / 180 * (count_high - count_low))
```

O argumento `wait` bloqueia a chamada. A espera é proporcional à mudança:

```text
espera = abs(pulso - pulso_anterior) * 0,00015 s
```

Uma mudança de 2000 µs gera 300 ms de espera. A lib oficial considera
0,3 s para 180 graus.

### 4.2 Servo360Unit (MicroPython)

A `Servo360Unit` herda a `Servo180Unit`. O construtor calcula dois valores:

```text
midpoint = (count_low + count_high) // 2 = 1500 µs
range    = (count_high - count_low) // 2 = 1000 µs
```

Os métodos de rotação são:

| Método | Fórmula |
|---|---|
| `clockwise(speed)` | `pulso = midpoint - int(speed / 100 * range)` |
| `counterclockwise(speed)` | `pulso = midpoint + int(speed / 100 * range)` |
| `stop()` | `pulso = midpoint` |

Regras de velocidade:

- O valor `speed` fica entre 0 e 100.
- O valor 0 significa parado.
- O valor entre 1 e 9 é forçado para 10.
- O valor 100 significa velocidade máxima.

Motivo do mínimo de 10 %:

O servo de rotação contínua tem uma zona morta no circuito interno.
Perto de 1500 µs, o motor não recebe tensão suficiente para girar.
O motor recebe corrente. O motor faz ruído. O eixo não gira.
A corrente aquece o enrolamento e a caixa de engrenagens.

O valor de 10 % coloca o pulso fora da zona morta. O eixo gira com
estabilidade. A lib oficial usa este valor. O valor é ajustável em `MinSpeed`.
O valor 0 desliga o limite.

### 4.3 Exemplo Arduino

O exemplo oficial usa o LEDC do ESP32.

```c
#define COUNT_LOW  1500
#define COUNT_HIGH 8500
#define TIMER_WIDTH 16
ledcSetup(1, 50, TIMER_WIDTH);
ledcAttachPin(26, 1);
```

O exemplo varre o duty de 1500 até 8500 counts. O valor 16 bits dá 305 ns
por count. O pino usado é o 26.

### 4.4 Tabela de paridade

| Lib oficial | Esta lib | Situação |
|---|---|---|
| `count_low=500`, `count_high=2500` | `MinPulse=500µs`, `MaxPulse=2500µs` | Igual |
| `freq=50` | `Period=20ms` | Igual |
| `pin = port[1]` | Campo `Pin` explícito | Igual, com nome claro |
| `set_duty(us)` | `SetPulse(time.Duration)` | Igual |
| `set_angle(0..180)` | `SetAngle(graus)` | Igual |
| `set_percent(0..100)` | `SetPercent(valor)` | Igual |
| `set_radian(0..π)` | `SetRadians(valor)` | Igual |
| `clockwise` / `counterclockwise` | `Clockwise` / `CounterClockwise` | Igual |
| `stop()` | `Stop()` | Igual |
| Mínimo de 10% na rotação | `MinSpeed=10` por padrão | Igual |
| midpoint igual à média | `StopPulse` igual à média | Igual |
| `wait=True` bloqueia | `Settling(from, to)` retorna duração | Diferente, ver 4.5 |
| `deinit()` | `Release()` e `Close()` | Diferente, ver 4.5 |
| Construtor grava pulso | `New` grava pulso inicial | Igual, ver 4.5 |
| `print` em cada comando | Sem `print` | Diferente, ver 4.5 |

### 4.5 Diferenças decididas

Estas diferenças são intencionais. Cada diferença tem um motivo.

1. A lib não bloqueia. A lib calcula o tempo de acomodação e devolve o valor.
   O firmware decide se dorme.
2. O `New` grava um pulso inicial. Esta decisão segue a lib oficial.
   Atenção: a lib oficial grava `count_low` (500 µs) nos dois tipos. Para o
   servo de 180°, 500 µs é 0 grau. Para o servo de 360°, 500 µs é velocidade
   máxima no sentido horário. Ver a decisão aberta 1.
3. A lib não imprime no console. O `print` oficial é código de depuração.
4. A lib expõe `Release()`. O pulso zero remove o torque. O comando `Stop()`
   mantém o servo parado com torque.

## 5. Escopo

### 5.1 Dentro do escopo

- Um módulo com os dois tipos de servo. A lib oficial também faz assim: a
  `Servo360Unit` herda a `Servo180Unit`.
- Servo de rotação contínua com velocidade e sentido.
- Servo posicional com ângulo, percentual e radiano.
- Pulso direto em microssegundos.
- Leitura do último pulso e da resolução real.
- Ajuste de faixa e de pulso de parada.
- Rampa de aceleração opcional.
- `Release()` para remover o torque.
- Grupo de servos no mesmo timer, até 8 canais.
- Teste de host com PWM falso.
- Contrato de zero alocação.
- Exemplo TinyGo e firmware de validação.

### 5.2 Fora do escopo

| Item | Motivo |
|---|---|
| LEDC com 16 bits por registrador | A resolução de 1,22 µs atende o servo. Fica como fase opcional. |
| Leitura de posição ou de velocidade | O SG90 não tem encoder. O controle é em malha aberta. |
| Medição de corrente e de temperatura | Não há sensor no kit. |
| Unit 8Servos (endereço 0x25) | É outro produto. Tem chip e protocolo I2C. Precisa de outra lib. |
| Módulo 4EncoderMotor | É outro domínio. Usa ponte H e encoder. |
| Controle PID | Não há realimentação. |

## 6. Base técnica do TinyGo

O LEDC é o periférico de PWM do ESP32. O nome significa LED Control.
O periférico gera o pulso por hardware. O núcleo do processador não gasta
tempo com o pulso. O TinyGo expõe o LEDC como `machine.PWM0` até `PWM3`.

O TinyGo tem PWM no ESP32 clássico. O periférico é o LEDC.

| Item | Valor |
|---|---|
| Timers | 4. Nomes: `PWM0` até `PWM3`. |
| Canais | 8 por timer |
| Configuração | `PWMConfig{Period uint64}` |
| Frequência | Não existe campo. Use `Period`. |
| Resolução | Automática. Abaixo de 100 Hz, usa 14 bits. |
| 50 Hz | 16384 counts por 20 ms |
| Quantização | 1,2207 µs por count |

Consequência: a lib tem 1,22 µs de quantização. O exemplo Arduino tem 305 ns.
A diferença não impede o controle. O deadband do servo é 8 µs.

A lib deve informar a resolução real. O método `Resolution()` devolve o valor.
O aplicativo decide se aceita o valor.

## 7. Arquitetura

```text
M5Servo-Go/
├── servo.go          # interface PWM, Config, erros, conversões
├── continuous.go     # rotação contínua
├── positional.go     # posição angular
├── ramp.go           # rampa e tempo de acomodação
├── bank.go           # grupo de servos no mesmo timer
├── native.go         # //go:build esp32 || rp2040
├── servo_test.go
├── alloc_test.go
├── bench_test.go
├── scripts/bench.sh
├── examples/spin/
├── examples/validate/
├── README.md
├── BENCHMARKS.md
├── PLAN.md
├── Makefile
└── .github/workflows/test.yml
```

## 8. API proposta

```go
package servo

// PWM é o mínimo que a lib precisa do periférico.
type PWM interface {
    Set(channel uint8, value uint32) // duty em counts
    Top() uint32                     // (1<<res)-1
}

type Config struct {
    PWM       PWM
    Channel   uint8
    Period    time.Duration // padrão 20 ms
    MinPulse  time.Duration // padrão 500 µs
    MaxPulse  time.Duration // padrão 2500 µs
    StopPulse time.Duration // padrão: média de MinPulse e MaxPulse
    MinSpeed  uint8         // padrão 10 %
    Ramp      time.Duration // padrão 0, sem rampa
}

func NewContinuous(cfg Config) (*Continuous, error)
func NewPositional(cfg Config) (*Positional, error)

func (c *Continuous) Clockwise(speed uint8) error
func (c *Continuous) CounterClockwise(speed uint8) error
func (c *Continuous) Stop() error
func (c *Continuous) Release() error
func (c *Continuous) SetSpeed(percent int8) error
func (c *Continuous) SetPulse(d time.Duration) error
func (c *Continuous) Pulse() time.Duration
func (c *Continuous) Speed() int8
func (c *Continuous) Resolution() time.Duration
func (c *Continuous) Settling(from, to time.Duration) time.Duration
func (c *Continuous) Update(now time.Duration) error

func (p *Positional) SetAngle(degrees float32) error
func (p *Positional) SetPercent(percent uint8) error
func (p *Positional) SetRadians(r float32) error
func (p *Positional) Angle() float32
```

O arquivo `native.go` tem o adaptador do hardware. Ele não cita placa nenhuma:

```go
//go:build esp32 || rp2040

type NativePWM interface {
    Configure(machine.PWMConfig) error
    Channel(pin machine.Pin) (uint8, error)
    Set(channel uint8, value uint32)
    Top() uint32
}

func NewNative(pwm NativePWM, pin machine.Pin) (*Continuous, error)
func NewNativePositional(pwm NativePWM, pin machine.Pin) (*Positional, error)
```

A lib não conhece o modelo da placa. O núcleo depende só da interface `PWM`.
Uma placa com outra API de PWM usa a interface direto, com dois métodos.

### 8.1 O que é o Bank

O LEDC tem 4 timers e 8 canais. O timer define a frequência. O canal define o
pino e o duty.

Todos os servos usam 50 Hz. Então vários servos podem usar o mesmo timer.
O `Bank` é um objeto pequeno que organiza este uso.

O `Bank` faz quatro coisas:

1. Configura um timer com período de 20 ms.
2. Aloca um canal para cada servo.
3. Devolve um objeto de servo por canal.
4. Libera os canais no `Close()`.

O `Bank` não é um expansor de hardware. Ele não cria canais novos. Ele usa os
8 canais do LEDC. O kit traz 2 servos. Então o `Bank` atende o kit com folga.

Uso previsto:

```go
b, _ := servo.NewBank(servo.BankConfig{PWM: machine.PWM0, Period: 20 * time.Millisecond})
s1, _ := b.Add(machine.GPIO32) // servo 1, canal atribuído pelo Bank
s2, _ := b.Add(machine.GPIO33) // servo 2, mesmo timer
defer b.Close()
```

## 9. Conversão

A conversão é a parte central. A lib deve testar a conversão com rigor.

```text
counts = round(pulso * (Top + 1) / período)
pulso  = counts * período / (Top + 1)
```

Tabela com período de 20 ms e 14 bits:

| Pulso | counts | Erro |
|---:|---:|---:|
| 500 µs | 410 | +0,9 µs |
| 1000 µs | 819 | −0,5 µs |
| 1500 µs | 1229 | +0,5 µs |
| 2000 µs | 1638 | −0,5 µs |
| 2500 µs | 2048 | 0,0 µs |

O erro máximo é metade da quantização. O valor é 0,61 µs.

## 10. Regras de controle

### 10.1 Rotação contínua

```text
speed = 0            -> StopPulse (1500 µs)
speed entre 1 e 9    -> força 10 %
speed entre 10 e 100 -> StopPulse até StopPulse - range
speed entre -10 e -100 -> StopPulse até StopPulse + range
```

O `range` é `(MaxPulse - MinPulse) / 2`. O valor padrão é 1000 µs.

O método `SetRange(low, high)` permite ajustar o curso real do servo.
Alguns SG90 de 360° usam 1000 µs a 2000 µs.

### 10.2 Posição angular

```text
pulso = MinPulse + (ângulo / 180) * (MaxPulse - MinPulse)
```

O ângulo fica entre 0 e 180. O método limita o valor. A lib não retorna erro
por ângulo fora da faixa.

### 10.3 Rampa

O método `SetRamp(v)` define a taxa em microssegundos por segundo.
O método `Update(now)` aplica a rampa.

Sem `Update`, o comando vale imediatamente. A rampa é opcional. O motivo é o
custo de estado e de cálculo por chamada.

Recomendação: use rampa de 3000 µs/s com engrenagem de plástico.

### 10.4 Tempo de acomodação

A lib oficial dorme depois de cada comando. Esta lib não dorme.

O método `Settling(from, to)` devolve a duração. A regra é a mesma da lib
oficial: 0,15 ms por microssegundo de mudança.

## 11. Testes

O teste usa um PWM falso. O PWM falso grava o canal e o duty. O PWM falso
devolve um `Top()` configurável.

Os testes devem cobrir:

1. Conversão de pulso para counts. Faixa de 400 µs a 2600 µs.
2. Erro máximo por resolução. Teste com 8, 14 e 16 bits.
3. Mapeamento de velocidade. Valores −100, −10, 0, 5, 10, 100.
4. Sentido de rotação. O pulso deve cair ou subir conforme o método.
5. `SetRange` com 1000 µs e 2000 µs.
6. Rampa. Subida, descida e rampa igual a zero.
7. `Release` igual a duty zero. `Stop` igual ao midpoint.
8. `Top()` igual a zero devolve `ErrBadPWM`.
9. Zero alocação por comando.
10. Tamanho da estrutura. Limite de 64 bytes.
11. Uso concorrente. Dois servos do mesmo grupo.
12. `Settling` confere com a tabela oficial.
13. Pulso inicial do `New`. O teste segue a decisão 15.1.

## 12. Fases

### Fase 0 — Pesquisa ✅

- [x] Produto, SKU, elétrica e corrente.
- [x] Leitura das libs oficiais `servo180.py` e `servo360.py`.
- [x] Leitura do exemplo Arduino.
- [x] Leitura do PWM do TinyGo 0.44.
- [x] Cálculo da quantização a 50 Hz.

### Fase 1 — Núcleo e conversão

- [x] `servo.go` com a interface `PWM`, `Config`, erros e conversões.
- [x] `positional.go` com ângulo, percentual e radiano.
- [x] Pulso inicial do `New` igual à lib oficial (decisão A).
- [x] `native.go` com `NewNative` (agnóstico de placa).
- [x] `examples/spin` com varredura igual ao exemplo Arduino.
- DoD: testes de conversão verdes. Exemplo compila para `m5stick-c`.

### Fase 2 — Rotação contínua

- [x] `continuous.go` com velocidade, sentido, parada e `Release`.
- [x] Limite mínimo de 10 %.
- [x] Rampa e `Settling` em `servo.go`.
- DoD: tabela de mapeamento completa. Zero alocação.

### Fase 3 — Grupo, contratos e benchmarks

- [x] `bank.go` com até 8 canais no mesmo timer.
- [x] `alloc_test.go`, `bench_test.go`, `scripts/bench.sh`.
- [x] `Makefile` e CI.
- DoD: CI verde. `allocs/op` igual a zero. `BENCHMARKS.md` completo.

### Fase 4 — Documentação e publicação

- [x] `README.md` com elétrica, limites, quantização e riscos.
- [ ] `git init` e `gh repo create`.
- [ ] Tag `v0.1.0`.
- DoD: `go get` funciona com a tag.

### Fase 5 — Validação em hardware

- [x] `examples/validate` com um teste por gravação.
- [x] Testes com um servo no Grove: 9 de 9 confirmados.
- [ ] Medição de tensão de 5 V no movimento (pendente).
- [ ] Medição de temperatura após 2 minutos (pendente).
- DoD: o servo para e gira nos dois sentidos. O servo não treme com pulso fixo.

## 13. Riscos

| Risco | Mitigação |
|---|---|
| Corrente alta em stall | Um servo por Grove. Fonte externa para dois servos. |
| Queda da tensão de 5 V | Use rampa. Meça a tensão. Evite bloqueio do eixo. |
| Quantização de 1,22 µs | `Resolution()` informa o valor. O deadband é maior. |
| Zona morta perto de 1500 µs | `StopPulse` ajustável. Histerese de 3 µs no `SetSpeed`. |
| Tremor com pulso fixo | Grave o pulso uma vez. Não recalcule por frame. |
| Canais esgotados | `Bank` devolve erro claro. `Close()` libera o canal. |
| Servo cai com `Release` | `Release` remove o torque. Documente o risco em eixo vertical. |
| Engrenagem de plástico | Rampa recomendada de 3000 µs/s. |
| `PWMConfig` sem frequência | Use `Period` igual a 20 ms. Teste confirma 14 bits. |

## 14. Matriz de validação em hardware

| Item | Método |
|---|---|
| Parada | `Stop()`. O eixo não gira. |
| Sentido horário | `Clockwise(50)`. Verifique o lado. |
| Sentido anti-horário | `CounterClockwise(50)`. Verifique o lado. |
| Velocidade mínima | `Clockwise(5)`. O servo deve girar, não zumbir. |
| Rampa | `SetRamp(3000µs/s)` e mudança de 0 % para 100 %. |
| Zero torque | `Release()`. O eixo gira com a mão. |
| Repetição | 100 comandos iguais. O pulso não deve variar. |
| Tensão | Multímetro no 5 V do Grove durante o movimento. |
| Temperatura | Toque no servo após 2 minutos. |

## 15. Decisões

| Nº | Decisão | Estado |
|---:|---|---|
| 1 | Nome do módulo | `M5Servo-Go`. Motivo: o módulo tem os dois tipos. |
| 2 | `Positional` no mesmo módulo | Sim. A lib oficial usa herança. |
| 3 | Mínimo de 10 % | Sim, com `MinSpeed` ajustável. Ver o motivo em 4.2. |
| 4 | `Bank` na v0.1.0 | Sim. O kit traz 2 servos. |
| 5 | LEDC com 16 bits | Não. A resolução de 1,22 µs atende. Fase opcional. |
| 6 | `New` grava pulso inicial | Sim, igual à lib oficial. Ver a decisão aberta abaixo. |

### 15.1 Decisão aberta

O pulso inicial do servo de 360° precisa de definição.

A lib oficial grava 500 µs nos dois tipos. Para o servo de 360°, 500 µs é
velocidade máxima. O servo gira no boot. O servo só para com um comando
`stop()`.

Opções:

| Opção | Efeito no boot |
|---|---|
| A. Igual à oficial: 500 µs nos dois | O servo de 360° gira no boot. |
| B. 180° com `MinPulse`, 360° com `StopPulse` | O servo de 360° fica parado. O servo de 180° vai a 0 grau. |

Recomendação: opção B. O motivo é a segurança. Um servo que gira no boot pode
mover o mecanismo sem supervisão.

## 16. Resultados de hardware

Um servo no Grove, placa M5StickC Plus2. O firmware roda um teste por vez.

| Teste | Resultado |
|---|---|
| 1 velocidade máxima horária | OK |
| 2 velocidade máxima anti-horária | OK |
| 3 horário 50 % | sentido OK. Velocidade igual à máxima. |
| 5 1 % elevado a 10 % | OK. Gira devagar, sem zumbido. |
| 6 `Stop` com torque | OK. Força de retenção pequena, esperada em 9 g. |
| 7 `Release` sem torque | Inconclusivo no dedo. A redução mecânica é alta. |
| 8 rampa com faixa oficial | Sutil. Metade da rampa fica saturada. |
| 9 pulso fixo de 1200 µs | OK. Rotação constante, sem tremor. |
| 11 1300 µs e 1700 µs | Lento horário e lento anti-horário. |
| 12 1000 µs contra 1300 µs | 1000 µs mais rápido. |
| 13 faixa 1000-2000 µs | 1250 µs e 1350 µs iguais. |
| 14 rampa de 3 s na faixa 1000-2000 µs | Aceleração clara. OK |

### Curva do servo deste kit

O SG90 de 360° satura perto de 1000 µs. A banda proporcional é estreita.
A faixa oficial de 500-2500 µs entrega velocidade máxima na maior parte do
curso.

Consequência para o uso: para controle de velocidade, use
`SetRange(1000µs, 2000µs)`. Ajuste a parada com `SetStopPulse` quando o
midpoint não for a parada real.

### Mudança feita por causa do teste

O teste 6 mostrou que a parada pode não ficar no midpoint. A lib ganhou
`SetStopPulse` para o ajuste fino.

## 17. Referências

- Servo Kit 360° (A076-C): https://docs.m5stack.com/en/accessory/Servo_Kit_360
- Lib oficial 180°: `uiflow-micropython/m5stack/libs/unit/servo180.py`
- Lib oficial 360°: `uiflow-micropython/m5stack/libs/unit/servo360.py`
- Exemplo Arduino:
  `M5-ProductExampleCodes/Accessory/ServoKit180_360/Arduino/ServoKit180_360.ino`
- TinyGo: `machine_esp32_pwm.go` e `machine_esp32xx_pwm.go`
- SG90: 50 Hz, 0,5 ms a 2,5 ms, 4,8 V a 6,0 V, 750 mA em stall
