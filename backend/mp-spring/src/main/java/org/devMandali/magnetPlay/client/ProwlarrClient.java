package org.devMandali.magnetPlay.client;

import org.devMandali.magnetPlay.model.SearchResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import org.w3c.dom.Document;
import org.w3c.dom.Element;
import org.w3c.dom.NodeList;
import reactor.core.publisher.Mono;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicLong;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Component
public class ProwlarrClient {

    private static final Logger log = LoggerFactory.getLogger(ProwlarrClient.class);
    private static final Pattern QUALITY = Pattern.compile(
        "(2160p|1080p|720p|480p|x265|x264|HEVC|BluRay|WEB-DL|WEBRip|HDRip|HDR|REMUX)",
        Pattern.CASE_INSENSITIVE);
    private static final Pattern API_KEY_RE = Pattern.compile("<ApiKey>([^<]+)</ApiKey>");

    private static final DocumentBuilderFactory DB_FACTORY;
    static {
        DB_FACTORY = DocumentBuilderFactory.newInstance();
        DB_FACTORY.setNamespaceAware(true);
        try {
            DB_FACTORY.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true);
            DB_FACTORY.setFeature("http://xml.org/sax/features/external-general-entities", false);
            DB_FACTORY.setFeature("http://xml.org/sax/features/external-parameter-entities", false);
            DB_FACTORY.setExpandEntityReferences(false);
        } catch (javax.xml.parsers.ParserConfigurationException e) {
            throw new ExceptionInInitializerError(e);
        }
    }

    @Value("${prowlarr.url}")
    private String prowlarrUrl;

    @Value("${prowlarr.api-key:}")
    private String configuredApiKey;

    @Value("${prowlarr.config-xml:../prowlarr-data/config.xml}")
    private String configXmlPath;

    private volatile String resolvedApiKey;
    private final WebClient webClient;

    public ProwlarrClient(WebClient.Builder builder) {
        this.webClient = builder.build();
    }

    public Mono<List<SearchResult>> search(String query) {
        String key = resolveApiKey();
        if (key == null || key.isBlank()) {
            return Mono.error(new IllegalStateException("Prowlarr API key unavailable — is Go sidecar running?"));
        }
        String baseUrl = prowlarrUrl.endsWith("/") ? prowlarrUrl.substring(0, prowlarrUrl.length() - 1) : prowlarrUrl;
        String url = baseUrl + "/api/v1/indexer/all/newznab?apikey=" + key
            + "&t=search&q=" + encode(query);

        return webClient.get()
            .uri(url)
            .retrieve()
            .bodyToMono(String.class)
            .map(this::parseAtomXml)
            .doOnError(e -> log.error("Prowlarr search error: {}", e.getMessage()));
    }

    private static final long RETRY_COOLDOWN_MS = 30_000;
    private final AtomicLong lastFailedAt = new AtomicLong(0);

    private String resolveApiKey() {
        String key = resolvedApiKey;
        if (key != null) return key;
        // Cooldown: don't hammer filesystem on every search call after a failure
        long failed = lastFailedAt.get();
        if (failed > 0 && System.currentTimeMillis() - failed < RETRY_COOLDOWN_MS) {
            return null;
        }
        synchronized (this) {
            if (resolvedApiKey != null) return resolvedApiKey;
            if (configuredApiKey != null && !configuredApiKey.isBlank()) {
                resolvedApiKey = configuredApiKey;
                return resolvedApiKey;
            }
            try {
                String xml = Files.readString(Path.of(configXmlPath));
                Matcher m = API_KEY_RE.matcher(xml);
                if (m.find()) {
                    resolvedApiKey = m.group(1);
                    lastFailedAt.set(0);
                    log.info("Prowlarr API key loaded from config.xml");
                    return resolvedApiKey;
                }
            } catch (IOException e) {
                log.warn("Cannot read prowlarr config.xml at {}: {}", configXmlPath, e.getMessage());
            }
            lastFailedAt.set(System.currentTimeMillis()); // retry after cooldown
            return null;
        }
    }

    private List<SearchResult> parseAtomXml(String xml) {
        List<SearchResult> results = new ArrayList<>();
        try {
            DocumentBuilder builder = DB_FACTORY.newDocumentBuilder();
            Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));
            NodeList items = doc.getElementsByTagName("item");
            for (int i = 0; i < items.getLength(); i++) {
                results.add(parseItem((Element) items.item(i)));
            }
        } catch (Exception e) {
            log.error("Torznab XML parse error: {}", e.getMessage());
        }
        return results;
    }

    private SearchResult parseItem(Element item) {
        String title = text(item, "title");
        String pubDate = text(item, "pubDate");
        String magnetUrl = "";
        long sizeBytes = 0;
        int seeders = 0;
        int peers = 0;
        String indexer = "";

        NodeList enclosures = item.getElementsByTagName("enclosure");
        if (enclosures.getLength() > 0) {
            Element enc = (Element) enclosures.item(0);
            String url = enc.getAttribute("url");
            if (url.startsWith("magnet:")) magnetUrl = url;
            String len = enc.getAttribute("length");
            if (!len.isBlank()) sizeBytes = parseLong(len);
        }

        // torznab:attr elements (namespace: http://torznab.com/schemas/2015/feed)
        NodeList attrs = item.getElementsByTagNameNS("http://torznab.com/schemas/2015/feed", "attr");
        for (int i = 0; i < attrs.getLength(); i++) {
            Element attr = (Element) attrs.item(i);
            String name = attr.getAttribute("name");
            String value = attr.getAttribute("value");
            switch (name) {
                case "seeders"   -> seeders = parseInt(value);
                case "peers"     -> peers = parseInt(value);
                case "size"      -> { if (sizeBytes == 0) sizeBytes = parseLong(value); }
                case "magneturl" -> { if (magnetUrl.isBlank()) magnetUrl = value; }
                case "indexer"   -> indexer = value;
            }
        }

        java.util.LinkedHashSet<String> tagSet = new java.util.LinkedHashSet<>();
        Matcher m = QUALITY.matcher(title);
        while (m.find()) {
            tagSet.add(m.group(1).toUpperCase());
        }
        List<String> qualityTags = new ArrayList<>(tagSet);

        return new SearchResult(title, sizeBytes, seeders, peers, indexer, pubDate, magnetUrl, qualityTags);
    }

    private String text(Element parent, String tag) {
        NodeList nl = parent.getElementsByTagName(tag);
        return nl.getLength() == 0 ? "" : nl.item(0).getTextContent();
    }

    private int parseInt(String s) {
        try { return Integer.parseInt(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private long parseLong(String s) {
        try { return Long.parseLong(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private String encode(String q) {
        return java.net.URLEncoder.encode(q, StandardCharsets.UTF_8);
    }
}
