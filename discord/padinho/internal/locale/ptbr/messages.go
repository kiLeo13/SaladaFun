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
	BirthdayNameLabel             = "🔖 Seu Nome"
	BirthdayNamePlaceholder       = "Seu nome ou apelido"
	BirthdayDateLabel             = "📅 Data de Nascimento"
	BirthdayDatePlaceholder       = "DD/MM/AAAA"
	BirthdayTimeZoneLabel         = "🕒 Fuso Horário"
	BirthdayButtonAdd             = "Adicionar"
	BirthdayButtonEdit            = "Editar"
	BirthdayInspectTitle          = "🔍 Dados de Aniversário"
	BirthdayEditDashboardTitle    = "✏️ Painel de Edição"
	BirthdayDashboardUserLabel    = "Usuário"
	BirthdayEditSelectPlaceholder = "Selecione o usuário que deseja editar"
	BirthdayUserIDLabel           = "ID do Usuário"
	BirthdayGuildUnknown          = "Servidor desconhecido"
	BirthdayNoRegistration        = "Este usuário não possui um aniversário cadastrado."
	BirthdaySelfNoRegistration    = "Você ainda não possui um aniversário cadastrado."
	BirthdayDefaultMessageValue   = "*Nenhuma mensagem personalizada*"
	BirthdayEditModalTitle        = "Editar %s"
	BirthdayEditSaved             = "Dados atualizados com sucesso."
	BirthdayAdministratorRequired = "Apenas administradores podem editar aniversários."
	BirthdayTimeZonePlaceholder   = "Selecione seu fuso horário"
	BirthdayTimeZoneBrasilia      = "Horário de Brasília (GMT-3)"
	BirthdayTimeZoneAmazonas      = "Horário do Amazonas (GMT-4)"
	BirthdayTimeZoneUTC           = "UTC"
	BirthdayMessageLabel          = "🧾 Mensagem Personalizada"
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
	GenericMessageCommandError    = "Algo deu errado ao processar esse comando."
	MoveAllCommandDescription     = "Move todos de um canal de voz para outro"
	MoveAllDestinationDescription = "Canal de voz de destino"
	MoveAllOriginDescription      = "Canal de voz de origem"
	MoveAllOriginRequired         = "Entre em um canal de voz ou informe o canal de origem."
	MoveAllInvalidOrigin          = "O canal de origem precisa ser um canal de voz."
	MoveAllInvalidDestination     = "O canal de destino precisa ser um canal de voz."
	MoveAllSameChannel            = "Os canais de origem e destino precisam ser diferentes."
	MoveAllStarted                = "Movendo %d membro(s)."
	OuroChestBalancedTitle        = "🎯 **Equilibrada — botão %d** (linha %d, coluna %d)"
	OuroChestInformationTitle     = "🔎 **Mais informação — botão %d** (linha %d, coluna %d)"
	OuroChestRewardTitle          = "💰 **Maior retorno — botão %d** (linha %d, coluna %d)"
	OuroChestRedTitle             = "🔴 **Chance imediata — botão %d** (linha %d, coluna %d)"
	OuroChestBalancedReason       = "Melhor compromisso: %s%% vermelho · %s bits de informação · EV %s esferas."
	OuroChestInformationReason    = "Maior redução de incerteza: %s bits · restam %s posições em média."
	OuroChestRewardReason         = "Maior valor imediato: EV %s esferas · %s%% vermelho."
	OuroChestRedReason            = "Maior chance de vermelho agora: %s%% · %s bits de informação."
	OuroChestCandidateFooter      = "\n\n-# %d posição(ões) possível(is) para o vermelho."
	OuroChestNoSuggestion         = "Não há um botão seguro disponível para recomendar."
	OuroChestInconsistent         = "Não consegui reconciliar as cores deste tabuleiro; não vou arriscar um palpite."
	OuroChestToggleUsage          = "Use `!toggleochelper` sem argumentos."
	OuroChestAutomaticEnabled     = "Assistência automática do `$oc` ativada. `!ochelper` continua disponível a qualquer momento."
	OuroChestAutomaticDisabled    = "Assistência automática do `$oc` desativada. Responda ao tabuleiro com `!ochelper` quando quiser ajuda."
	OuroChestManualUsage          = "Responda a um tabuleiro do Mudae usando `!ochelper`."
	OuroChestManualNotMudae       = "A mensagem respondida não foi enviada pelo Mudae."
	OuroChestManualNotBoard       = "A mensagem respondida não contém um tabuleiro 5×5 válido."
	OuroChestManualNotOC          = "Não consegui confirmar que essa mensagem é um tabuleiro de `$oc`."
	OuroChestManualFinished       = "Esse tabuleiro de `$oc` já terminou."
	OuroChestManualActive         = "A assistência desse tabuleiro já está ativa."
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
