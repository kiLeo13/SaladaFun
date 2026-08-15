// Package ptbr contains Padinho's user-facing Brazilian Portuguese copy.
package ptbr

const (
	BirthdayCommandDescription  = "Lista os aniversários do servidor"
	BirthdayTitle               = "Aniversários de %s"
	BirthdayEmptyMonth          = "Nenhum aniversário cadastrado neste mês."
	BirthdayEntry               = "**%02d/%02d** — %s"
	BirthdayAddModalTitle       = "Adicionar meu aniversário"
	BirthdayNameLabel           = "Como você quer ser chamado?"
	BirthdayNamePlaceholder     = "Seu nome ou apelido"
	BirthdayDateLabel           = "Data de nascimento"
	BirthdayDatePlaceholder     = "AAAA-MM-DD"
	BirthdayTimeZoneLabel       = "Fuso horário"
	BirthdayTimeZonePlaceholder = "Ex.: America/Sao_Paulo"
	BirthdayMessageLabel        = "Mensagem de aniversário"
	BirthdayMessagePlaceholder  = "Use {age}, {name} e {mention} se quiser"
	BirthdaySaved               = "Seu aniversário foi salvo! 🎉"
	BirthdayInvalidName         = "Informe um nome ou apelido com até 100 caracteres."
	BirthdayInvalidDate         = "Informe a data de nascimento no formato AAAA-MM-DD."
	BirthdayInvalidTimeZone     = "Informe um fuso horário IANA válido, como America/Sao_Paulo."
	BirthdayInvalidMessage      = "A mensagem aceita somente os campos {age}, {name} e {mention}."
	BirthdayOnlyOwner           = "Somente quem abriu a lista pode trocar a página."
	BirthdayInvalidInteraction  = "Este botão não é mais válido. Execute /birthdays novamente."
	GenericInteractionError     = "Algo deu errado ao processar essa interação."
	BirthdayDefaultMessage      = "Feliz aniversário, {mention}! Hoje {name} completa {age} anos! 🎉"
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
