package main

var markdownHTMLHeader = string(`<!DOCTYPE html>
<html class="no-js" lang="en">
    <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width" />
        <meta name="author" content="Colin Morris">
        <title>Personal site for Colin Morris/ Professor von Explaino. Contains code, steampunk and nonsense.</title>
        <link rel="preload" as="style" href="https://vonexplaino.com/theme/blog/style/blog.css" />
        <link rel="stylesheet" href="https://vonexplaino.com/theme/blog/style/blog.css" />
        <link rel="alternate" type="application/rss+xml" title="Professor von Explaino's Journal RSS Feed" href="/blog/rss.xml" />
        <link rel="pingback" href="https://webmention.io/vonexplaino.com/xmlrpc">
        <link rel="webmention" href="https://webmention.io/vonexplaino.com/webmention">
        <link rel="icon" type="image/svg+xml" href="https://vonexplaino.comfavicon.svg">
        <link rel="icon" type="image/png" href="https://vonexplaino.comfavicon.png">
        <meta name="theme-color" content="#ffffff" />
        <meta name="description" content="Personal site for Colin Morris/ Professor von Explaino. Code and steampunk thoughts/ experimentation" />
    </head>
    <body class="container">
        <main id="professor-paper" class="panel">
        <div id="top-left" class="decorative-tops"></div>
        <div id="top-right" class="decorative-tops"></div>
        <div id="bottom-left" class="decorative-tops"></div>
        <div id="bottom-right" class="decorative-tops"></div>
        <section id="introduction" class="h-card">`)

var markdownHTMLFooter = string(`</section>
</body>
</html>
`)
