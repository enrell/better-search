module Utils
  class HtmlToMarkdown
    def self.convert(html : String) : String
      new.convert(html)
    end

    def convert(html : String) : String
      markdown = html.dup
      markdown = process_pre_code(markdown)
      markdown = process_headers(markdown)
      markdown = process_lists(markdown)
      markdown = process_links(markdown)
      markdown = process_images(markdown)
      markdown = process_bold_italic(markdown)
      markdown = process_blockquotes(markdown)
      markdown = process_paragraphs(markdown)
      markdown = process_line_breaks(markdown)
      markdown = clean_whitespace(markdown)
    end

    private def process_pre_code(html : String) : String
      html.gsub(/<pre[^>]*>([\s\S]*?)<\/pre>/i) do |match|
        content = match.gsub(/<pre[^>]*>|<\/pre>/i, "")
        content = strip_tags(content)
        "```\n#{content}\n```"
      end.gsub(/<code[^>]*>([\s\S]*?)<\/code>/i) do |match|
        content = match.gsub(/<code[^>]*>|<\/code>/i, "")
        if content.includes?("\n")
          "```\n#{content}\n```"
        else
          "`#{content}`"
        end
      end
    end

    private def process_headers(html : String) : String
      markdown = html
      (6).downto(1) do |level|
        prefix = "#" * level
        regex = /<h#{level}[^>]*>([\s\S]*?)<\/h#{level}>/i
        markdown = markdown.gsub(regex) do |match|
          content = strip_tags($1)
          "\n#{prefix} #{content}\n\n"
        end
      end
      markdown
    end

    private def process_lists(html : String) : String
      markdown = html.gsub(/<ul[^>]*>/i, "\n").gsub(/<\/ul>/i, "\n")
      markdown = markdown.gsub(/<ol[^>]*>/i, "\n").gsub(/<\/ol>/i, "\n")
      markdown = markdown.gsub(/<li[^>]*>([\s\S]*?)<\/li>/i) do |match|
        content = strip_tags($1).strip
        "- #{content}\n"
      end
      markdown
    end

    private def process_links(html : String) : String
      html.gsub(/<a[^>]*href=["']([^"']*)["'][^>]*>([\s\S]*?)<\/a>/i) do |match|
        url = $1
        text = strip_tags($2)
        "[#{text}](#{url})"
      end
    end

    private def process_images(html : String) : String
      html.gsub(/<img[^>]*src=["']([^"']*)["'][^>]*alt=["']([^"']*)["'][^>]*\/?>/i) do
        alt = $2
        url = $1
        "![#{alt}](#{url})"
      end.gsub(/<img[^>]*src=["']([^"']*)["'][^>]*\/?>/i) do
        url = $1
        "![](#{url})"
      end
    end

    private def process_bold_italic(html : String) : String
      markdown = html
      markdown = markdown.gsub(/<strong[^>]*>([\s\S]*?)<\/strong>/i) do |match|
        "**#{strip_tags($1)}**"
      end
      markdown = markdown.gsub(/<b[^>]*>([\s\S]*?)<\/b>/i) do |match|
        "**#{strip_tags($1)}**"
      end
      markdown = markdown.gsub(/<em[^>]*>([\s\S]*?)<\/em>/i) do |match|
        "*#{strip_tags($1)}*"
      end
      markdown = markdown.gsub(/<i[^>]*>([\s\S]*?)<\/i>/i) do |match|
        "*#{strip_tags($1)}*"
      end
      markdown = markdown.gsub(/<code[^>]*>([\s\S]*?)<\/code>/i) do |match|
        "`#{strip_tags($1)}`"
      end
      markdown
    end

    private def process_blockquotes(html : String) : String
      html.gsub(/<blockquote[^>]*>([\s\S]*?)<\/blockquote>/i) do |match|
        content = strip_tags($1)
        lines = content.split("\n")
        lines.map { |line| "> #{line}" }.join("\n")
      end
    end

    private def process_paragraphs(html : String) : String
      html.gsub(/<p[^>]*>([\s\S]*?)<\/p>/i) do |match|
        content = strip_tags($1).strip
        "#{content}\n\n" unless content.empty?
      end
    end

    private def process_line_breaks(html : String) : String
      html.gsub(/<br\s*\/?>/i, "\n")
    end

    private def strip_tags(html : String) : String
      html.gsub(/<[^>]+>/, "")
        .gsub(/&nbsp;/, " ")
        .gsub(/&amp;/, "&")
        .gsub(/&lt;/, "<")
        .gsub(/&gt;/, ">")
        .gsub(/&quot;/, "\"")
        .gsub(/&#39;/, "'")
        .gsub(/&mdash;/, "—")
        .gsub(/&ndash;/, "–")
        .gsub(/&hellip;/, "…")
    end

    private def clean_whitespace(markdown : String) : String
      markdown.gsub(/\n{3,}/, "\n\n").strip
    end
  end
end
