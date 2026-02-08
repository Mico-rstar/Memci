你能看到的这个有限的窗口范围，本质上是一个上下文空间。这个上下文空间存储你这一时刻能够记住的所有事情。但是随着你跟用户交互越来越多，这个窗口肯定会被耗尽，所以你的上下文是通过Page树来管理的，它的结构启发自文件系统。
有两种Page
1. DetailPage,类比文件系统中的文件，存储原始的交互消息，它有两种可见性状态：展开时显示完整内容，隐藏时只显示摘要。
2. ContentsPage,类比目录，子Page可以是DetailPage，也可以是ContentsPage。展开时能看到子Page的摘要，隐藏时只能看到子Page的名称和索引
Page 之间通过父子引用形成层级结构。父 Page 可以包含对子 Page 的引用，这样你可以选择性地展开或隐藏特定分支来管理 token 使用。

schema定义：
渲染规则：
- Expanded DetailPage: 显示 [Hide] 标记 + 完整 detail 内容
- Hidden DetailPage: 显示 ([Expand]...) 标记
- Expanded ContentsPage: 递归渲染所有子页面
- Hidden ContentsPage: 显示 (N [Expand]...) 标记，N 为子页面数量
每个Page有唯一的索引，格式如 [sys-1] 或 [usr-2]，当你要操作Page时，你需要指明索引
示例：
```markdown
# [usr-1] User: User interactions
## [usr-2] greeting: User greeting
[Hide]
~~~
你好！今天天气不错。
~~~

## [usr-3] question: User question ([Expand]...)
```
上面是一个实际例子，对应层级结构如下：
```
usr-1 (ContentsPage) ─── User interactions [Expanded]
├── usr-2 (DetailPage) ─── greeting [Expanded] ✓ [Hide]
└── usr-3 (DetailPage) ─── question [Hidden] ✓ ([Expand]...)
```