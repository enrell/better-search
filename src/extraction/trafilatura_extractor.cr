require "lexbor"

module Extraction
  class TrafilaturaExtractor
    REMOVE_TAGS = [
      "script", "style", "nav", "aside", "footer", "header",
      "form", "iframe", "noscript", "svg", "button", "input",
      "select", "textarea", "menu", "figure", "figcaption",
    ]

    BOOST_CLASSES   = ["content", "article", "post", "entry", "main", "text", "body"]
    PENALTY_CLASSES = ["comment", "sidebar", "footer", "header", "nav", "menu", "widget", "ad", "advertisement", "social", "share"]

    def extract(html : String) : ExtractionResult
      doc = Lexbor::Parser.new(html)

      remove_unwanted_tags(doc)
      metadata = extract_metadata(doc)
      main_content = find_main_content(doc)
      cleaned = clean_content(main_content)

      ExtractionResult.new(
        title: metadata[:title],
        text: cleaned,
        author: metadata[:author],
        date: metadata[:date],
        language: metadata[:language],
        url: metadata[:url]
      )
    end

    private def remove_unwanted_tags(doc : Lexbor::Parser)
      REMOVE_TAGS.each do |tag|
        doc.nodes(tag).each do |node|
          node.remove!
        end
      end
    end

    private def extract_metadata(doc : Lexbor::Parser) : Metadata
      title = extract_title(doc)
      author = extract_author(doc)
      date = extract_date(doc)
      language = extract_language(doc)
      url = extract_url(doc)

      {title: title, author: author, date: date, language: language, url: url}
    end

    private def extract_title(doc : Lexbor::Parser) : String
      title = ""

      doc.nodes("meta[property=og:title]").each do |meta|
        title = meta["content"]? || ""
        break if title
      end

      if title.empty?
        title_node = doc.nodes("title").first?
        title = title_node.inner_text if title_node
      end

      if title.empty?
        h1 = doc.nodes("h1").first?
        title = h1.inner_text if h1
      end

      title.strip
    end

    private def extract_author(doc : Lexbor::Parser) : String
      author = ""

      doc.nodes("meta[name=author]").each do |meta|
        author = meta["content"]? || ""
        break if author
      end

      if author.empty?
        author = doc.nodes("author").first?.try(&.inner_text) || ""
      end

      author.strip
    end

    private def extract_date(doc : Lexbor::Parser) : String
      date = ""

      doc.nodes("meta[property=article:published_time]").each do |meta|
        date = meta["content"]? || ""
        break if date
      end

      if date.empty?
        time_node = doc.nodes("time").first?
        if time_node
          date = time_node["datetime"]? || time_node.inner_text
        end
      end

      date.strip
    end

    private def extract_language(doc : Lexbor::Parser) : String
      lang = ""

      html_node = doc.nodes("html").first?
      lang = html_node["lang"]? || "" if html_node

      if lang.empty?
        html_node = doc.nodes("html").first?
        lang = html_node["xml:lang"]? || "" if html_node
      end

      lang.strip
    end

    private def extract_url(doc : Lexbor::Parser) : String
      url = ""

      doc.nodes("meta[property=og:url]").each do |meta|
        url = meta["content"]? || ""
        break if url
      end

      if url.empty?
        link = doc.nodes("link[rel=canonical]").first?
        url = link["href"]? || "" if link
      end

      url.strip
    end

    private def find_main_content(doc : Lexbor::Parser) : String
      candidates = [] of {node: Lexbor::Node, score: Float64}

      article = doc.nodes("article").first?
      if article
        content = article.inner_text
        score = calculate_score(article)
        candidates << {node: article, score: score}
      end

      main = doc.nodes("main").first?
      if main
        candidates << {node: main, score: calculate_score(main)}
      end

      role_main = doc.nodes("[role=main]").first?
      if role_main
        candidates << {node: role_main, score: calculate_score(role_main)}
      end

      doc.nodes("div").each do |div|
        div_id = div["id"]?
        div_class = div["class"]?
        next if div_id.nil? && div_class.nil?

        class_id = "#{div_id} #{div_class}".downcase

        boost = BOOST_CLASSES.any? { |c| class_id.includes?(c) }
        penalty = PENALTY_CLASSES.any? { |c| class_id.includes?(c) }

        next if penalty && !boost

        score = calculate_score(div)
        score += 2.0 if boost
        score -= 2.0 if penalty

        candidates << {node: div, score: score}
      end

      if candidates.empty?
        return doc.body.try(&.inner_text) || ""
      end

      best = candidates.max_by { |c| c[:score] }
      best[:node].inner_text rescue ""
    end

    private def calculate_score(node : Lexbor::Node) : Float64
      text = node.inner_text rescue ""
      links = extract_link_text(node)

      return 0.0 if text.empty?

      text_length = text.bytesize
      link_text_length = links.sum(&.bytesize)

      return 0.0 if text_length == 0

      link_density = link_text_length.to_f / text_length.to_f
      text_density = 1.0 - link_density

      score = text_density * 10.0
      score += Math.log(text_length) / 2.0

      score
    end

    private def extract_link_text(node : Lexbor::Node) : Array(String)
      links = [] of String
      node.nodes("a").each do |a|
        links << a.inner_text rescue ""
      end
      links
    end

    private def clean_content(text : String) : String
      cleaned = text.dup
      cleaned = cleaned.gsub(/\s+/, " ")
      cleaned = cleaned.gsub(/\n\s*\n/, "\n\n")
      cleaned.strip
    end
  end

  record ExtractionResult,
    title : String,
    text : String,
    author : String,
    date : String,
    language : String,
    url : String

  alias Metadata = NamedTuple(
    title: String,
    author: String,
    date: String,
    language: String,
    url: String)
end
