// Package ptbr contains Padinho's user-facing Brazilian Portuguese copy.
package ptbr

const (
	BirthdayCommandDescription    = "Lista os aniversários do servidor"
	BirthdayTitle                 = "🥳 %s"
	BirthdayEmptyMonth            = "Nenhum aniversário para este mês."
	BirthdayEntry                 = "**%02d/%02d** — %s"
	BirthdayAddModalTitle         = "Adicionar Aniversário"
	BirthdayNameLabel             = "Nome do Usuário"
	BirthdayNamePlaceholder       = "Seu nome ou apelido"
	BirthdayDateLabel             = "Data de Nascimento"
	BirthdayDatePlaceholder       = "AAAA-MM-DD"
	BirthdayTimeZoneLabel         = "Fuso horário"
	BirthdayButtonAdd             = "Adicionar"
	BirthdayTimeZonePlaceholder   = "Ex.: America/Sao_Paulo"
	BirthdayMessageLabel          = "Mensagem de aniversário"
	BirthdayMessagePlaceholder    = "Use {age}, {name} e {mention} se quiser"
	BirthdaySaved                 = "Seu aniversário foi salvo! 🎉"
	BirthdayInvalidName           = "Informe um nome ou apelido com até 100 caracteres."
	BirthdayInvalidDate           = "Informe a data de nascimento no formato AAAA-MM-DD."
	BirthdayInvalidTimeZone       = "Informe um fuso horário IANA válido, como America/Sao_Paulo."
	BirthdayInvalidMessage        = "A mensagem aceita somente os campos {age}, {name} e {mention}."
	BirthdayInvalidMonth          = "Selecione um mês válido entre janeiro e dezembro."
	BirthdayManageServerRequired  = "Você precisa da permissão Gerenciar servidor para adicionar aniversários."
	BirthdayInvalidInteraction    = "Este botão não é mais válido. Execute /birthdays novamente."
	GenericInteractionError       = "Algo deu errado ao processar essa interação."
	BirthdayDefaultMessage        = "Feliz aniversário, {mention}! Hoje {name} completa {age} anos! 🎉"
	MoveAllCommandDescription     = "Move todos de um canal de voz para outro"
	MoveAllDestinationDescription = "Canal de voz de destino"
	MoveAllOriginDescription      = "Canal de voz de origem"
	MoveAllOriginRequired         = "Entre em um canal de voz ou informe o canal de origem."
	MoveAllInvalidOrigin          = "O canal de origem precisa ser um canal de voz."
	MoveAllInvalidDestination     = "O canal de destino precisa ser um canal de voz."
	MoveAllSameChannel            = "Os canais de origem e destino precisam ser diferentes."
	MoveAllStarted                = "Movendo %d membro(s)."
)

var MonthNames = [...]string{
	"",
	"janeiro",
	"fevereiro",
	"março",
	"abril",
	"maio",
	"junho",
	"julho",
	"agosto",
	"setembro",
	"outubro",
	"novembro",
	"dezembro",
}
