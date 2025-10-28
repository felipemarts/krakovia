package blockchain

import (
	"encoding/json"
	"fmt"
)

// TransactionType define os tipos de transação suportados
type TransactionType string

const (
	TransactionTypeTransfer TransactionType = "transfer" // Transferência simples de tokens
	TransactionTypeStake    TransactionType = "stake"    // Depositar tokens em stake
	TransactionTypeUnstake  TransactionType = "unstake"  // Sacar tokens do stake
	TransactionTypeData     TransactionType = "data"     // Dados arbitrários (para futuro)
)

// TransactionData representa os dados específicos de uma transação
type TransactionData struct {
	Type    TransactionType    `json:"type"`              // Tipo da transação
	Payload map[string]interface{} `json:"payload,omitempty"` // Dados específicos do tipo
}

// NewTransferData cria dados para uma transação de transferência
func NewTransferData() *TransactionData {
	return &TransactionData{
		Type:    TransactionTypeTransfer,
		Payload: make(map[string]interface{}),
	}
}

// NewStakeData cria dados para uma transação de stake
func NewStakeData(amount uint64) *TransactionData {
	return &TransactionData{
		Type: TransactionTypeStake,
		Payload: map[string]interface{}{
			"amount": amount,
		},
	}
}

// NewUnstakeData cria dados para uma transação de unstake
func NewUnstakeData(amount uint64) *TransactionData {
	return &TransactionData{
		Type: TransactionTypeUnstake,
		Payload: map[string]interface{}{
			"amount": amount,
		},
	}
}

// NewCustomData cria dados customizados (para extensibilidade futura)
func NewCustomData(dataType string, payload map[string]interface{}) *TransactionData {
	return &TransactionData{
		Type:    TransactionType(dataType),
		Payload: payload,
	}
}

// GetString retorna um valor string do payload
func (td *TransactionData) GetString(key string) (string, bool) {
	if td.Payload == nil {
		return "", false
	}
	val, ok := td.Payload[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetUint64 retorna um valor uint64 do payload
func (td *TransactionData) GetUint64(key string) (uint64, bool) {
	if td.Payload == nil {
		return 0, false
	}
	val, ok := td.Payload[key]
	if !ok {
		return 0, false
	}

	// Tenta conversão de diferentes tipos numéricos
	switch v := val.(type) {
	case uint64:
		return v, true
	case float64:
		return uint64(v), true
	case int:
		return uint64(v), true
	case int64:
		return uint64(v), true
	default:
		return 0, false
	}
}

// GetBool retorna um valor bool do payload
func (td *TransactionData) GetBool(key string) (bool, bool) {
	if td.Payload == nil {
		return false, false
	}
	val, ok := td.Payload[key]
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// SetValue define um valor no payload
func (td *TransactionData) SetValue(key string, value interface{}) {
	if td.Payload == nil {
		td.Payload = make(map[string]interface{})
	}
	td.Payload[key] = value
}

// Serialize serializa TransactionData para JSON
func (td *TransactionData) Serialize() (string, error) {
	if td == nil {
		return "", nil
	}
	data, err := json.Marshal(td)
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction data: %w", err)
	}
	return string(data), nil
}

// DeserializeTransactionData desserializa TransactionData de JSON
func DeserializeTransactionData(data string) (*TransactionData, error) {
	if data == "" {
		return nil, nil
	}

	// Tenta fazer unmarshal como JSON
	var td TransactionData
	err := json.Unmarshal([]byte(data), &td)
	if err != nil {
		// Se falhar, assume que é um texto simples (para compatibilidade)
		// Retorna nil (sem TransactionData estruturado)
		return nil, nil
	}
	return &td, nil
}

// Validate valida os dados da transação baseado no tipo
func (td *TransactionData) Validate() error {
	if td == nil {
		return nil // Transação sem dados é válida
	}

	switch td.Type {
	case TransactionTypeTransfer:
		// Transfer não precisa de validações extras (Amount já está na Transaction)
		return nil

	case TransactionTypeStake:
		amount, ok := td.GetUint64("amount")
		if !ok {
			return fmt.Errorf("stake transaction missing amount in payload")
		}
		if amount == 0 {
			return fmt.Errorf("stake amount must be greater than 0")
		}
		return nil

	case TransactionTypeUnstake:
		amount, ok := td.GetUint64("amount")
		if !ok {
			return fmt.Errorf("unstake transaction missing amount in payload")
		}
		if amount == 0 {
			return fmt.Errorf("unstake amount must be greater than 0")
		}
		return nil

	case TransactionTypeData:
		// Dados arbitrários - sem validação específica
		return nil

	default:
		// Tipos customizados - permite extensibilidade
		return nil
	}
}

// IsStakeOperation verifica se é uma operação de stake
func (td *TransactionData) IsStakeOperation() bool {
	if td == nil {
		return false
	}
	return td.Type == TransactionTypeStake || td.Type == TransactionTypeUnstake
}

// GetStakeAmount retorna o amount de uma operação de stake/unstake
func (td *TransactionData) GetStakeAmount() (uint64, error) {
	if !td.IsStakeOperation() {
		return 0, fmt.Errorf("not a stake operation")
	}

	amount, ok := td.GetUint64("amount")
	if !ok {
		return 0, fmt.Errorf("stake amount not found or invalid")
	}

	return amount, nil
}

// Clone cria uma cópia profunda de TransactionData
func (td *TransactionData) Clone() *TransactionData {
	if td == nil {
		return nil
	}

	clone := &TransactionData{
		Type:    td.Type,
		Payload: make(map[string]interface{}),
	}

	for k, v := range td.Payload {
		clone.Payload[k] = v
	}

	return clone
}
