package context

import (
	"fmt"
	"memci/message"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPageStorage(t *testing.T) {
	storage, err := NewFilePageStorage("./test/storage", false)
	if err != nil {
		t.Errorf("failed to create page storage: %v", err)
	}
	page := NewPage(PageIndex(0), "test_page", -1, "this is a test page", nil)
	page.AddEntry(NewEntry(message.System, message.NewContentString("111")))
	page.AddEntry(NewEntry(message.System, message.NewContentString("222")))
	page.AddEntry(NewEntry(message.System, message.NewContentString("333")))

	// 验证 head 不为 nil
	require.NotNil(t, page.Head())

	err = storage.Save(page, page.Index)
	if err != nil {
		t.Errorf("failed to save page: %v", err)
	}

	loadedPage, err := storage.Load(page.Index)
	if err != nil {
		t.Errorf("failed to load page: %v", err)
	}

	// 验证基本字段
	require.Equal(t, page.Index, loadedPage.Index)
	require.Equal(t, page.Name, loadedPage.Name)
	require.Equal(t, page.Description, loadedPage.Description)

	// 验证 Entry 数量
	originalEntries := page.GetEntries()
	loadedEntries := loadedPage.GetEntries()
	require.Equal(t, len(originalEntries), len(loadedEntries))

	// 验证每个 Entry 的 Node 数据是否正确序列化
	for i := 0; i < len(originalEntries); i++ {
		originalEntry := originalEntries[i]
		loadedEntry := loadedEntries[i]

		// 验证 ID
		require.Equal(t, originalEntry.ID, loadedEntry.ID)

		// 验证 Node 的消息数据（深拷贝）
		originalMsg := originalEntry.Node.GetMsg()
		loadedMsg := loadedEntry.Node.GetMsg()

		require.Equal(t, originalMsg.Role, loadedMsg.Role)
		require.Equal(t, originalMsg.Content.String(), loadedMsg.Content.String())

		// 验证 Node 是不同的对象（深拷贝）
		require.NotSame(t, originalEntry.Node, loadedEntry.Node)

		// 验证反序列化后的 Node 的链表连接已重建
		if i > 0 {
			require.Equal(t, loadedEntries[i-1].Node, loadedEntry.Node.GetPrev(),
				"Node %d prev should point to previous entry's Node", i)
		} else {
			require.Nil(t, loadedEntry.Node.GetPrev(), "First Node's prev should be nil")
		}

		if i < len(originalEntries)-1 {
			require.Equal(t, loadedEntries[i+1].Node, loadedEntry.Node.GetNext(),
				"Node %d next should point to next entry's Node", i)
		} else {
			require.Nil(t, loadedEntry.Node.GetNext(), "Last Node's next should be nil")
		}

	}
}

func TestRebuildLinkedList(t *testing.T) {
	page1 := NewPage(PageIndex(0), "page1", -1, "this is a page 1", nil)
	page1.AddEntry(NewEntry(message.System, message.NewContentString("111")))
	page1.AddEntry(NewEntry(message.System, message.NewContentString("222")))
	page1.AddEntry(NewEntry(message.System, message.NewContentString("333")))
	page2 := NewPage(PageIndex(1), "page2", -1, "this is a page 2", nil)
	page2.AddEntry(NewEntry(message.System, message.NewContentString("444")))
	page2.AddEntry(NewEntry(message.System, message.NewContentString("555")))
	page2.AddEntry(NewEntry(message.System, message.NewContentString("666")))

	testChapter := NewBaseChapter(TypeSystemChapter)
	testChapter.AddPage(page1, page1.Index)
	testChapter.AddPage(page2, page2.Index)

	msgList := testChapter.ToMessageList()

	// 使用便捷遍历方法验证链表结构
	count := 0
	msgList.ForEachNode(func(node *message.MessageNode) {
		fmt.Printf("Node %d: %s\n", count, node.GetMsg().Content.String())
		count++
	})
	require.Equal(t, 6, count, "Should have 6 nodes in total")

	storage, err := NewFilePageStorage("./test/storage", false)
	if err != nil {
		t.Errorf("failed to create page storage: %v", err)
	}

	err = storage.Save(page2, page2.Index)
	if err != nil {
		t.Errorf("failed to save page: %v", err)
	}

	fmt.Println()

	count = 0
	page2.Detach()

	msgList.ForEachNode(func(node *message.MessageNode) {
		fmt.Printf("Node %d: %s\n", count, node.GetMsg().Content.String())
		count++
	})
	require.Equal(t, 3, count, "Should have 3 nodes after Detach")
}
