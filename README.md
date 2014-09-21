# MakeEpub

此工具可以将 *html* 文件转换为 *epub* 格式的电子书。它根据html文件中的特定标签，将其拆分为章节，并自动生成目录等信息。

This tool helps to create *epub* books from *html* files. It split the html file into chapters according to special tags in the file and generate the TOC automatically.

这些特定标签被称为“拆分点”，有“标题标签(\<h1\>, \<h2\>,...\<h6\>)”和“章节标签(*class* 属性包含 *makeepub-chapter* 的标签)”两种。章节标签主要在需要对章节标题进行修饰时使用，如在章节标题前插入横幅图片等。

The special tags are call "split point", including two kinds of tags: "header tags(\<h1\>, \<h2\>, ... \<h6\>)" and "chapter tags (tag's *class* attribute contains *makeepub-chapter* )". Chapter tags is useful if chapter title decoration is required, for example: add a banner image before the title.

每个拆分点有“级别”和“标题”两个属性，分别对应目录中的级别和标题。其中级别是一个0到6之间的整数，但0级拆分点只用于文件拆分，不会出现在目录中。

Every split point has two properties: 'level' and 'title', they are mapped to the level and title properties of a TOC item. 'Level' is an integer betwwen 0 and 6, but level 0 split point is only for file split, will not be used for generate TOC.

此工具支持批处理模式，可一次性转换生成多个epub文件。它还支持打包、解包epub文件，合并html文件或文本文件；还可以作为一个web服务器，转换上传的zip文件为epub文件。

The tool support batch mode which can generate multiple epub books in one execution. It also support pack/extract an epub book, merge html/text files. And can be used as a web server to convert an uploaded zip file to an epub book.

## 1. 命令行(Command Line)

	转换(Create)       : makeepub <VirtualFolder> [OutputFolder] [-epub2] [-noduokan]
	批处理(Batch)      : makeepub -b <InputFolder> [OutputFolder] [-epub2] [-noduokan]
                         makeepub -b <BatchFile> [OutputFolder] [-epub2] [-noduokan]
	打包(Pack)         : makeepub -p <VirtualFolder> <OutputFile>
	解包(Extract)      : makeepub -e <EpubFile> <OutputFolder>
	合并(Merge) HTML   : makeepub -mh <VirtualFolder> <OutputFile>
	合并(Merge) Text   : makeepub -mt <VirtualFolder> <OutputFile>
	Web服务器(Server)  : makeepub -s [Port]

各参数含义如下：

The meaning of the arguments are as below:

+ **VirtualFolder** : 一个文件夹(如example文件夹下的book文件夹)或zip文件(如example文件夹下的book.zip)，里面包含要处理的文件。(An OS folder (for example: folder *book* in folder *example*) or a zip file(for example: *book.zip* in folder *example*) which contains the input files.)
+ **OutputFolder** 一个文件夹，用于保存输出文件。(An OS folder to store the output file(s).)
+ **InputFolder**  : 一个文件夹，里面有输入文件或文件夹。(An OS folder which contains the input folder(s)/file(s).)
+ **-epub2** : 默认生成EPUB3格式的文件，使用此参数将生成EPUB2格式的文件。(By default, the output file is EPUB3 format, use this argument if EPUB2 format is required.)
+ **-noduokan** : 禁用 [多看](http://www.duokan.com/) 扩展。(Disable [DuoKan](http://www.duokan.com/) externsion.)
+ **BatchFile**    : 一个文本文件，里面列出了所有要处理的VirtualFolder，每行一个。(A text which lists the path of 'VirtualFolders' to be processed, one line for one 'VirtualFolder'.)
+ **OutputFile**   : 输出文件的路径。(The path of the output file.)
+ **EpubFile**     : 一个epub文件的路径。(The path of an EPUB file.)
+ **Port**         : Web服务器的监听端口，默认80。(The TCP port for the web server to listen to, default value is 80.)

## 2. 转换(Create)

	makeepub <VirtualFolder> [OutputFolder]

处理 *VirtualFolder* 中的文件，生成epub，并保存到 *OutputFolder* 中。在VirtualFolder中，必须有以下三个文件：

Process files in *VirtualFolder*, generate epub file and save it to *OutputFolder* . The 3 files below 3 are mandatory and must exist in VirtualFolder:

+ **book.ini** 配置文件，用于指定书名、作者等信息(configuration file to specify book name, author and etc.)
+ **book.html** 书的正文(The content of the book)
+ **cover.png** or **cover.jpg** or **cover.gif** 封面图片文件(The cover image of the book)

请 **务必** 使用 *UTF-8* 编码保存前两个文件，否则程序可能不能正确处理。

The first 2 files **MUST** stored in *UTF-8* encoding, otherwise, the tool may not able to process them correctly.

除以上文件外，其它书籍需要的文件，如层叠样式表（css），图片等也应保存到此文件夹中。如果文件内容是文本，建议也使用 *UTF-8* 编码保存。

Besides the 3 files above, other files required by the book, like sytle sheet (css) and images, are 
also required to be put into the folder. And if the content of a file is text, it is also recommend to store it in *UTF-8* encoding.

### 2.1 文件格式

下面是对主要文件的格式的简单介绍，更多信息请参考example文件夹中的示例。

Below is a brief introduction of the format of the mandatory files, more information please refer to the examples in the *example* folder.

#### book.ini

此文件基于通用的ini文件格式，以'='开始的行将被合并到上一行，以'#'开始的行将被视为注释，并被忽略。

This file is based on the common *INI* file format, line start with '=' will be joint to previous line, and line start with '#' will be regard as comment and ignored.

这个文件包含三个节， *book* 、 *split* 和 *output*，book节指定书籍信息，split节指定如何进行章节拆分，output节指定输出文件信息。下面的列表将介绍其中每一个选项的作用。

This file contains three sections: *book*, *split* and *output*. section *book* is for the book information, *split* determines how chapters are split, and section *output* is for the output file. The below list explains the usage of each option.

+ Book节(Section Book)
	- **name**: 书名，如果没有提供会导致程序输出一个警告信息(Name of the book, if not specified, the tool will generate a warning)
	- **author**:  作者，如果没有提供会导致程序输出一个警告信息(Author of the book, if not specified, the tool will generate a warning)
	- **id**: 书的唯一标识，在正规出版的书中，它应该是ISBN编号，如果您没有指定，程序将随机生成一个(The unique identifier, it is the ISBN for a published book. If not specified, the tool will generate a random string for it.)
	- **publisher**: 出版社(The publisher of the book.)
	- **description**: 书籍简介(A brief introduction of the book.)
	- **language**: 语言，默认 *zh-CN* ，即简体中文(Language of the book, *zh-CN* by default, that's Chinese Simplified.)
	- **toc**: 一个 *1* 到 *6* 之间的整数，用于指定目录的粒度，默认为 *2*，即只生成1、2两级拆分点对应的目录(An integer between *1* and *6*, specifis how to TOC is generated. Default value is *2*, which means the TOC is based on level 1 and level 2 split points)

+ Split节(section Split)
	- **AtLevel**: 一个 *0* 到 *6* 之间的整数，用于指定章节拆分的粒度，默认为 *1*，即只根据1级拆分点拆分章节(An integer between *0* and *6*, specifis how to split the html file into chapters. Default value is *1*, which means the split is based on the level 1 split points)
	- **ByHeader**: 一个 *1* 到 *7* 之间的整数。如果一个“标题标签”拆分点的级别小于此选项的值，那么这个拆分点将被忽略。默认值是1，即不忽略任何“标题标签”拆分点。(An integer between *1* and *7*. A "header" split point will be ignored if its level property is smaller than this value. Default is *1* which means no "header" split point will be ignored.)
	
+ Output节(Section Output)
	- **path**: 输出epub文件的路径。如果没有指定，程序会产生一个警告且不会生成任何文件(The output path of the target epub file. If the path is not specified, the tool will generate a warning and no file will be created)

下面是book.ini的一个例子。

Below is an example for book.ini.

	[book]
	name=My First eBook
	author=Super Man
	id=ISBN XXXXXXXXXXXX
	publisher=My Own Press
	description= 这是本书的简介，它占用了多行。 This is the description
	           = of the book, and it has more than one line.
	language=zh-CN
	toc=2
	
	[split]
	AtLevel=1
	ByHeader=1
	
	[output]
	path=d:\MyBook.epub


#### book.html

它是一个标准的html文件，根据 *split* 节的设置，程序会将此文件拆分成章节文件，根据 *toc* 设置生成书籍目录。\<body\>标签之前的内容会被复制到每个章节文件的开头。

This is a standard html file. The tool will split this file into chapter files based on *split* setting, and generate TOC based on the *toc* setting. Content before \<body\> tag will be copied to the beginning of each chapter file.

如果其中的某个 *img* 标签符合以下情况，它将会全屏显示 (An image is displayed as full screen if its *img* tag meet all below conditions):
+ 打开了多看扩展 (DuoKan externsion is enabled)
+ *img* 标签的父级是 *body* 标签 (The parent of *img* tag is *body* tag)
+ *img* 的 *class* 属性包含 *duokan-fullscreen* (The value of the *class* property of the *img* tag contains *duokan-fullscreen* )

#### cover.png/jpg/gif

一个图片文件，它将被用来生成封面。这个文件可以是cover.png、cover.jpg和cover.gif中的任意一个，如果存在多个，如同时有cover.png和cover.jpg，那么程序会随机使用其中一个生成封面。

An image file which will be used to create the book cover. It can be 'cover.png', 'cover.jpg' or 'cover.gif', if more than one file exists (for example: both 'cover.png' and 'cover.jpg'), the tool will select one randomly.

封面文件的名字是cover.html，所以请勿使用这个文件名，否则程序的行为将是未知的。

The file name of the cover page is 'cover.html', please don't use this name for any other purpose, otherwise the behavior of this tool is not defined.

### 2.2 拆分点(Split Point)

拆分点的使用遵循以下规则，具体使用方法请参考 *example* 文件夹中的例子。

Below are the rules of split point, please refer to the *example* folder for examples.

0. 所有拆分点都必须是 *body* 标签的直接子标签。(All split point MUST be the direct child of the *body* tag.)
1. 默认情况下，“标题标签”都是拆分点，其“级别”是这个标签的级别，“标题”是这个标签的内容。 (By default, all "header tags" are split points, their "level" are the level of the tags and "title" are the content of these tags.)
2. “标题标签”的“标题”也可以通过 *data-chapter-title* 属性指定，这种情况下，目录中的标题和正文中的标题将不一样。("Title" can also be specified by *data-chapter-title* attribute, in this case, the chapter will have different title in TOC and content.)
3. 如果一个标题标签的 *class* 属性包含 *makeepub-not-chapter* ，那么它不是拆分点。(A header tag is not split point when its *class* attribute contains *makeepub-not-chapter* .)
4. 任何标签，如果它的 *class* 属性包含 *makeepub-chapter* ，那么它是一个“章节标签”拆分点。(A tag is a "chapter tag" split point if its *class* attribute contains *makeepub-chapter* .)
5. “章节标签”拆分点的“级别”和“标题”可以由 *data-chapter-level* 和 *data-chapter-title* 属性指定。(The "level" and "title" of a "chapter tag" can be specified by the "data-chapter-level" and "data-chapter-title" attributes.)
6. 如果一个“章节标签”没有 *data-chapter-level* 属性，那么它的“级别”和“标题”由后续的（包括此标签）第一个“标题标签”决定，同时这个“标题标签”失效。但在找到所需的“标题标签”之前，如果出现了其他“章节标签”，则此“章节标签”失效。(If a "chapter tag" does not have *data-chapter-level* attribute, its "level" and "title" will be determined by the first "header tag" after it (or itself, if it is a "header tag" also), and the "header tag" will be ignored. But, if another "chapter tag" is found before the required "header tag", this "chapter tag" will be ignored.)
7. “章节标签”的优先级高于“标题标签”，即如果一个标签既是“章节标签”又是“标题标签”，它将被作为“章节标签”处理。(The priority of "chapter tag" is higer than "header tag", so if a tag is both "chapter tag" and "header tag", it is regarded as "chapter tag".)
8. 0级拆分点只用于文件拆分，不生成目录。(Level 0 split point is only for file split, will not be used for generate TOC.)
9. 级别小于 *ByHeader* 的“标题标签”拆分点会全部被忽略。("Header tag" split points whose level are smaller than *ByHeader* will be ignored.)
10. 级别大于 *toc* 的拆分点不会生成目录。(Split points whose level are larger than *toc* will not appear in TOC.)
11. 级别大于 *AtLevel* 的拆分点不会造成文件拆分。(File split will not happen on split points whose level are larger than *AtLevel* .)
12. 为尽量避免拆分出来的文件只包含章节标题，即使某个拆分点按照 *AtLevel* 选项应该被拆分，如果它和它的上级拆分点之间没有任何正文，它也不会被拆分。(To avoid a chapter file only has a chapter title, file split will not happen on a split point if there's no text between the split point and its parent split point, no matter what the value of option *AtLevel* is.)

### 2.3 输出文件的路径(path of the output file)

输出文件的路径取决于 *book.ini* 中 *output* 节的 *path* 选项，命令行中的 *OutputFolder* 参数，以及程序的当前工作文件夹。

The path of the out file is determined by the *path* option of section *output* in *book.ini*, argument *OutputFolder* in command line, and the current working folder of the tool.

如果缺少path选项，不会生成任何文件。

No file will be create if there's no option *path*.

如果没有OutputFolder参数，且path是相对路径，输出文件路径是相对于当前工作文件夹的path。

If argument *OutputFolder* is not specified, and *path* is relative, the output file path will be *path* relative to the current working folder.

如果没有OutputFolder参数，且path是绝对路径，输出文件路径是path。

If argument *OutputFolder* is not specified, and *path* is absolute, the output file path is *path*.

如果指定了OutputFolder，文件会被保存在OutputFolder，文件名是path中的文件名部分。

If *OutputFolder* is specified, output file will be save at *OutputFolder*, and file name is the 'file name' in *path*.

## 3. 批处理(Batch)

	makeepub -b <InputFolder> [OutputFolder] [-epub2] [-noduokan]
	makeepub -b <BatchFile> [OutputFolder] [-epub2] [-noduokan]

批处理模式，相当于对InputFolder中的(或BatchFile列出的)每个VirtualFolder **folder**，调用：

Batch mode, is equal to: for each *VirtualFolder* **folder** in *InputFolder* (or listed in *BatchFile), call:

	makeepub folder [OutputFolder] [-epub2] [-noduokan]
	

## 4. 打包(Pack)

	makeepub -p <VirtualFolder> <OutputFile>

将VirtualFolder中的文件打包成一个EPUB文件，保存为OutputFile。

Pack the files in *VirtualFolder* into an EPUB and save it as *OutputFile*.

## 5. 解包(Extract)

	makeepub -e <EpubFile> <OutputFolder>

将EpubFile解包到OutputFolder中。

Extract *EpubFile* to folder *OutputFolder*.

## 6. 合并(Merge)

	makeepub -mh <VirtualFolder> <OutputFile>
	makeepub -mt <VirtualFolder> <OutputFile>

按文件名升序合并VirtualFolder中的文件，并将合并结果保存为OutputFile。合并模式可以使html模式(-mh)或文本(-mt)。

Sort files in *VirtualFolder* in ascend order by file name, merge them, and save the merge result as *OutputFile*. The merge mode can be *html*(-mh) or *text*(-mt).

文本模式是简单的将文件内容连接在一起，Html模式会分析文件，只保留一份文件头(&lt;body&gt;之前的部分)和文件尾(&lt;/body&gt;之后的部分)。

*text* mode is simply merge file content one by one. *html* mode will analysis the file to keep only one copy of file header (content before &lt;body&gt;) and file footer (content after &lt;/body&gt;).


## 7. Web服务器(Web Server)

	makeepub -s [Port]

以Web服务器形式运行，处理用户上传的zip文件，生成EPUB文件，供用户下载。

Run as a web server, process the user uploaded zip file, and generate the EPUB file for user to download.

如果不需要此功能，可将其删除以减小可执行文件的体积。

If you don't need this feature, it can be removed to reduce the size of the executable file.

## 8. 授权及其他(License & Others)

MakeEpub是自由软件，基于[MIT授权](http://opensource.org/licenses/mit-license.html)发布

MakeEpub is free software distributed under the terms of the [MIT license](http://opensource.org/licenses/mit-license.html).

此程序是根据我自己制作epub书籍的需要编写，同时也通过编写过程熟悉了[Go语言](http://golang.org/)(可能需翻墙)。今后，将仅修正bug，而不再增加新的功能。

This tool is developed for my own need when creating epub book, I also learned the [Go program language](http://golang.org/). From now on, I will only fix bugs and won't add new feature any more.