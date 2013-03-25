#MakeEpub

此工具用于将 *html* 文件转换为 *epub* 格式。它根据html文件中的 *\<h1\>*， *\<h2\>* 等标签，将其拆分为章节，并自动生成目录等信息。

This tool helps to convert *html* file to *epub* format. It split the html file into chapters by the tags *\<h1\>*, *\<h2\>* ..., and generate the TOC automatically.

##1. 使用方法(Usage)

	makeepub folder [output]

其中 *folder* 是一个文件夹，包含要转换的html文件及其它文件。 *output* 用于指定目标epub文件的路径，此参数可选。

*folder* is used to store the html file and other files that need to convert. *output* is the path of the target epub file, and it is optional.

##2. 源文件(Source Files)

所有源文件都要放在命令行参数中指定的 *folder* 文件夹中，以下三个源文件是必须的：

All source files must be put into the *folder* in the command line argument, and below 3 files are mandatory:

+ **book.ini** 配置文件，用于指定书名、作者等信息(configuration file to specify book name, author and etc.)
+ **book.html** 书的正文(The content of the book)
+ **cover.html** 封面(The cover of the book)

请 **务必** 使用 *UTF-8* 编码保存这三个文件，否则程序可能不能正确处理。

The 3 files **MUST** stored in *UTF-8* encoding, otherwise, the tool may not able to process them correctly.

除以上文件外，其它书籍需要的文件，如层叠样式表（css），图片等也应保存到此文件夹中。如果文件内容是文本，建议也使用 *UTF-8* 编码保存。

Besides the 3 files above, other files required by the book, like sytle sheet (css) and images, are 
also required to be put into the folder. And if the content of a file is text, it is also recommend to store it in *UTF-8* encoding.

##3. 文件格式(File Format)

这里是对主要文件的格式进行一个简单的介绍，更多信息请参考example文件夹中的示例。

Here is a brief introduction of the format of the mandatory files, more information please refer to the example book in the *example* folder.

###3.1 book.ini

此文件基于通用的ini文件格式，每一行的行首不能有空白字符，以'#'开始的行将被视为注释，并被忽略。

This file is based on the common *INI* file format, blank characters are not allowed at the beginning of a line, and line start with '#' will be regard as comment and ignored.

这个文件包含两个节， *book* 和 *output*，book节指定书籍信息，output节指定输出文件信息。下面的列表将介绍其中每一个选项的作用。

This file contains two sections: *book* and *output*. section *book* is for the book information, while section *output* is for the output file. The below list explains the usage of each option.

+ Book节(Section Book)
	- **name**: 书名，如果没有提供会导致程序输出一个警告信息(Name of the book, if not specified, the tool will generate a warning)
	- **author**:  作者，如果没有提供会导致程序输出一个警告信息(Author of the book, if not specified, the tool will generate a warning)
	- **id**: 书的唯一标识，在正规出版的书中，它应该是ISBN编号，如果您没有指定，程序将随机生成一个(The unique identifier, it is the ISBN for a published book. If not specified, the tool will generate a random string for it.)
	- **depth**: 一个 *1* 到 *6* 之间的整数，用于指定章节拆分的粒度，默认为 *1*，即只根据 *h1* 标签拆分章节(An integer between *1* and *6*, specifis how to split the html file into chapters. Default value is *1*, which means the split is based on the *h1* tags)
	
+ Output节(Section Output)
	- **path**: 输出epub文件的路径。如果命令行中有 *output* 参数，这个选项会被忽略。如果这里和命令行都没有指定输出路径，程序会产生一个警告(The output path of the target epub file. If the *output* argument exists in the command line, this option will be ignored. If the path hasn't been specified either in the command line nor here, the tool will generate a warning) 

下面是book.ini的一个例子。

Below is an example for book.ini.

	[book]
	name=My First eBook
	author=Super Man
	id=ISBN XXXXXXXXXXXX
	depth=1
	
	[output]
	path=d:\MyBook.epub


###3.2 book.html

它是一个标准的html文件，但必须保证以下两点：

This is a standard html file, but the 2 points below must be followed:

+ \<body\>标签必须独占一行 (The \<body\> tag must be in its own line)
+ “\<h1\>...\</h1\>”等标题必须独占一行(Headers like "\<h1\>...\</h1\>" must be in their own lines)

根据 *depth* 设置，程序会将此文件拆分成章节文件，\<body\>标签之前的内容会被复制到每个章节文件的开头。

According to the *depth* setting, the tool will split this file into chapter files, the content before \<body\> tag will be copied to the beginning of each chapter file.


###3.3 cover.html

它是一个标准的html文件，一般情况下您直接拷贝下面示例文件的内容就可以了，根据实际情况，您可能需要修改一下图片文件（cover.jpg）的名字。

This is a standard html file. In most case, you can simply copy the content of the example file listed below. According to the scenario, you may need to change the name of the image file (cover.jpg).


	<?xml version='1.0' encoding='utf-8'?>
	<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
	    <head>
	        <meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
	        <meta name="calibre:cover" content="true"/>
	        <title>Cover</title>
	        <style type="text/css" title="override_css">
	            @page {padding: 0pt; margin:0pt}
	            body { text-align: center; padding:0pt; margin: 0pt; }
	        </style>
	    </head>
	    <body>
	        <div>
	            <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="100%" height="100%" viewBox="0 0 600 800" preserveAspectRatio="none">
	                <image width="600" height="800" xlink:href="cover.jpg"/>
	            </svg>
	        </div>
	    </body>
	</html>


##4. 授权(License)

MakeEpub是自由软件，基于[MIT授权](http://opensource.org/licenses/mit-license.html)发布

MakeEpub is free software distributed under the terms of the [MIT license](http://opensource.org/licenses/mit-license.html).
