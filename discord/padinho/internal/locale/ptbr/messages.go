// Package ptbr contains Padinho's user-facing Brazilian Portuguese copy.
package ptbr

const (
	BirthdayCommandDescription    = "Lista os aniversários do servidor"
	BirthdayTitle                 = "🥳 %s"
	BirthdayEmptyMonth            = "Nenhum aniversário para este mês."
	BirthdayEntry                 = "**%02d/%02d** — <@%d>"
	BirthdayUpcoming              = "-# Próximo aniversário: <@%d> <t:%d:R>"
	BirthdayNoUpcoming            = "Nenhum aniversário cadastrado."
	BirthdayAddModalTitle         = "Adicionar Aniversário"
	BirthdayUserLabel             = "Usuário do Aniversário"
	BirthdayUserPlaceholder       = "Selecione o usuário"
	BirthdayNameLabel             = "Nome do Usuário"
	BirthdayNamePlaceholder       = "Seu nome ou apelido"
	BirthdayDateLabel             = "Data de Nascimento"
	BirthdayDatePlaceholder       = "DD/MM/AAAA"
	BirthdayTimeZoneLabel         = "Fuso Horário"
	BirthdayButtonAdd             = "Adicionar"
	BirthdayTimeZonePlaceholder   = "Selecione seu fuso horário"
	BirthdayTimeZoneBrasilia      = "Horário de Brasília (GMT-3)"
	BirthdayTimeZoneAmazonas      = "Horário do Amazonas (GMT-4)"
	BirthdayTimeZoneUTC           = "UTC"
	BirthdayMessageLabel          = "Mensagem de Aniversário"
	BirthdayMessagePlaceholder    = "Use {age}, {name} e {mention} se quiser"
	BirthdaySaved                 = "Seu aniversário foi salvo! 🎉"
	BirthdaySavedForUser          = "Aniversário de <@%d> salvo!"
	BirthdayInvalidName           = "Informe um nome ou apelido com até 100 caracteres."
	BirthdayInvalidDate           = "Informe a data de nascimento no formato DD/MM/AAAA."
	BirthdayInvalidTimeZone       = "Informe um fuso horário IANA válido, como America/Sao_Paulo."
	BirthdayInvalidMessage        = "A mensagem aceita somente os campos {age}, {name} e {mention}."
	BirthdayInvalidMonth          = "Selecione um mês válido entre janeiro e dezembro."
	BirthdayManageServerRequired  = "Você precisa da permissão Gerenciar servidor para adicionar aniversários."
	BirthdayInvalidInteraction    = "Este botão não é mais válido. Execute /birthdays novamente."
	GenericInteractionError       = "Algo deu errado ao processar essa interação."
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
