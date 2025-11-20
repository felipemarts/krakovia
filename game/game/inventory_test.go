package game

import (
	"testing"
)

// TestBlockTypeSize verifica se BlockType suporta IDs customizados
func TestBlockTypeSize(t *testing.T) {
	// BlockType deve ser uint16 para suportar IDs >= 256
	var bt BlockType = 257
	if bt != 257 {
		t.Errorf("BlockType deve suportar valores >= 256, got %d", bt)
	}

	// CustomBlockIDStart deve ser 256
	if CustomBlockIDStart != 256 {
		t.Errorf("CustomBlockIDStart deve ser 256, got %d", CustomBlockIDStart)
	}

	// Verificar IsCustomBlock
	if !IsCustomBlock(BlockType(257)) {
		t.Error("IsCustomBlock(257) deve retornar true")
	}

	if IsCustomBlock(BlockType(1)) {
		t.Error("IsCustomBlock(1) deve retornar false")
	}

	// DefaultBlockID (256) is a custom block now
	if !IsCustomBlock(BlockType(DefaultBlockID)) {
		t.Error("IsCustomBlock(BlockType(DefaultBlockID)) deve retornar true")
	}
}

// TestCustomBlockManager verifica o gerenciador de blocos customizados
func TestCustomBlockManager(t *testing.T) {
	cbm := NewCustomBlockManager()

	// Criar um bloco
	block := cbm.CreateBlock("TestBlock")
	if block == nil {
		t.Fatal("CreateBlock retornou nil")
	}

	if block.Name != "TestBlock" {
		t.Errorf("Nome do bloco incorreto: got %s, want TestBlock", block.Name)
	}

	if block.ID < CustomBlockIDStart {
		t.Errorf("ID do bloco deve ser >= %d, got %d", CustomBlockIDStart, block.ID)
	}

	// Verificar se o bloco pode ser recuperado
	retrieved := cbm.GetBlock(block.ID)
	if retrieved == nil {
		t.Error("GetBlock retornou nil para ID válido")
	}

	if retrieved.Name != block.Name {
		t.Errorf("GetBlock retornou bloco com nome incorreto: got %s, want %s", retrieved.Name, block.Name)
	}

	// Listar blocos
	blocks := cbm.ListBlocks()
	if len(blocks) != 1 {
		t.Errorf("ListBlocks deve retornar 1 bloco, got %d", len(blocks))
	}
}

// TestGetBlockName verifica se os nomes dos blocos são retornados corretamente
func TestGetBlockName(t *testing.T) {
	cbm := NewCustomBlockManager()

	// Criar o bloco padrão (não falha mesmo sem a textura)
	cbm.EnsureDefaultBlock()

	// Criar um bloco customizado
	block := cbm.CreateBlock("MeuBloco")

	// Criar inventário mockado (sem hotbar real)
	inv := &InventoryUI{
		CustomBlockMgr: cbm,
	}

	// Testar nome de bloco padrão (agora é um bloco customizado com nome "Default")
	name := inv.getBlockName(BlockType(DefaultBlockID))
	if name != "Default" {
		t.Errorf("Nome do BlockType(DefaultBlockID) incorreto: got %s, want Default", name)
	}

	// Testar nome de bloco customizado
	name = inv.getBlockName(BlockType(block.ID))
	if name != "MeuBloco" {
		t.Errorf("Nome do bloco customizado incorreto: got %s, want MeuBloco", name)
	}
}

// TestInventoryRefreshBlockList verifica se a lista de blocos é atualizada corretamente
func TestInventoryRefreshBlockList(t *testing.T) {
	cbm := NewCustomBlockManager()

	// Criar o bloco padrão (não falha mesmo sem a textura)
	cbm.EnsureDefaultBlock()

	// Criar alguns blocos customizados
	cbm.CreateBlock("Bloco1")
	cbm.CreateBlock("Bloco2")
	cbm.CreateBlock("Bloco3")

	// Criar inventário mockado
	inv := &InventoryUI{
		CustomBlockMgr:  cbm,
		FilteredBlocks:  make([]BlockType, 0),
	}

	// Atualizar lista
	inv.refreshBlockList()

	// Deve ter 4 blocos (Default + 3 customizados)
	if len(inv.FilteredBlocks) != 4 {
		t.Errorf("FilteredBlocks deve ter 4 blocos, got %d", len(inv.FilteredBlocks))
	}

	// Primeiro bloco deve ser Default (ID 256)
	if inv.FilteredBlocks[0] != BlockType(DefaultBlockID) {
		t.Errorf("Primeiro bloco deve ser BlockType(DefaultBlockID), got %d", inv.FilteredBlocks[0])
	}

	// Todos os blocos devem ser customizados (ID >= 256)
	for i := 0; i < len(inv.FilteredBlocks); i++ {
		if !IsCustomBlock(inv.FilteredBlocks[i]) {
			t.Errorf("Bloco %d deve ser customizado, ID: %d", i, inv.FilteredBlocks[i])
		}
	}
}

// TestInventoryLimit30Blocks verifica se o inventário limita a 30 blocos
func TestInventoryLimit30Blocks(t *testing.T) {
	cbm := NewCustomBlockManager()

	// Criar 35 blocos customizados
	for i := 0; i < 35; i++ {
		cbm.CreateBlock("Bloco")
	}

	// Criar inventário mockado
	inv := &InventoryUI{
		CustomBlockMgr:  cbm,
		FilteredBlocks:  make([]BlockType, 0),
	}

	// Atualizar lista
	inv.refreshBlockList()

	// Deve estar limitado a 30 blocos
	if len(inv.FilteredBlocks) > 30 {
		t.Errorf("FilteredBlocks deve ter no máximo 30 blocos, got %d", len(inv.FilteredBlocks))
	}
}

// TestInventorySearch verifica se a pesquisa funciona
func TestInventorySearch(t *testing.T) {
	cbm := NewCustomBlockManager()

	// Criar alguns blocos com nomes específicos
	cbm.CreateBlock("Pedra")
	cbm.CreateBlock("Madeira")
	cbm.CreateBlock("Pedregulho")

	// Criar inventário mockado
	inv := &InventoryUI{
		CustomBlockMgr:  cbm,
		FilteredBlocks:  make([]BlockType, 0),
		SearchBuffer:    "madeira",
	}

	// Atualizar lista com filtro
	inv.refreshBlockList()

	// Deve encontrar apenas "Madeira" (1 bloco)
	if len(inv.FilteredBlocks) != 1 {
		t.Errorf("Pesquisa por 'madeira' deve retornar 1 bloco, got %d", len(inv.FilteredBlocks))
	}

	// Testar pesquisa que retorna múltiplos resultados
	inv.SearchBuffer = "pedr"
	inv.refreshBlockList()

	// Deve encontrar "Pedra" e "Pedregulho" (2 blocos)
	if len(inv.FilteredBlocks) != 2 {
		t.Errorf("Pesquisa por 'pedr' deve retornar 2 blocos, got %d", len(inv.FilteredBlocks))
		// Debug: mostrar os blocos encontrados
		for i, bt := range inv.FilteredBlocks {
			t.Logf("Bloco %d: %s (ID: %d)", i, inv.getBlockName(bt), bt)
		}
	}
}
