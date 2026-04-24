package org.devMandali.magnetPlay.client;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.devMandali.magnetPlay.model.SearchResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.ExchangeStrategies;
import org.springframework.web.reactive.function.client.WebClient;
import org.w3c.dom.Document;
import org.w3c.dom.Element;
import org.w3c.dom.NodeList;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Component
public class ProwlarrClient {

    private static final Logger log = LoggerFactory.getLogger(ProwlarrClient.class);
    private static final Pattern QUALITY = Pattern.compile(
        "(2160p|1080p|720p|480p|x265|x264|HEVC|BluRay|WEB-DL|WEBRip|HDRip|HDR|REMUX)",
        Pattern.CASE_INSENSITIVE);
    private static final Pattern MAGNET_RE = Pattern.compile(
        "magnet:\\?xt=urn:btih:[A-Za-z0-9]+[^\\s<\"']*");
    private static final Pattern API_KEY_RE = Pattern.compile("<ApiKey>([^<]+)</ApiKey>");

    private static final ObjectMapper MAPPER = new ObjectMapper();

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
    private volatile Map<Integer, String> cachedIndexers; // id → name
    private final WebClient webClient;

    public ProwlarrClient(WebClient.Builder builder) {
        this.webClient = builder
            .exchangeStrategies(ExchangeStrategies.builder()
                .codecs(c -> c.defaultCodecs().maxInMemorySize(10 * 1024 * 1024))
                .build())
            .build();
    }

    public Mono<List<SearchResult>> search(String query) {
        String key = resolveApiKey();
        if (key == null || key.isBlank()) {
            return Mono.error(new IllegalStateException("Prowlarr API key unavailable — is Go sidecar running?"));
        }
        String baseUrl = prowlarrUrl.endsWith("/") ? prowlarrUrl.substring(0, prowlarrUrl.length() - 1) : prowlarrUrl;

        return fetchIndexers(baseUrl, key)
            .flatMap(indexers -> {
                if (indexers.isEmpty()) {
                    log.warn("No Prowlarr indexers configured — seeding may still be in progress");
                    return Mono.just(List.<SearchResult>of());
                }
                return Flux.fromIterable(indexers.entrySet())
                    .flatMap(e -> queryIndexerNewznab(baseUrl, key, e.getKey(), e.getValue(), query))
                    .collectList()
                    .map(lists -> lists.stream()
                        .flatMap(List::stream)
                        .distinct()
                        .sorted(java.util.Comparator.comparingInt(SearchResult::seeders).reversed())
                        .collect(java.util.stream.Collectors.toList()));
            })
            .doOnError(e -> log.error("Prowlarr search error: {}", e.getMessage()));
    }

    private Mono<Map<Integer, String>> fetchIndexers(String baseUrl, String key) {
        Map<Integer, String> cached = cachedIndexers;
        if (cached != null && !cached.isEmpty()) return Mono.just(cached);
        return webClient.get()
            .uri(baseUrl + "/api/v1/indexer?apikey=" + key)
            .retrieve()
            .bodyToMono(String.class)
            .map(json -> {
                Map<Integer, String> indexers = new LinkedHashMap<>();
                try {
                    JsonNode root = MAPPER.readTree(json);
                    for (JsonNode node : root) {
                        int id = node.path("id").asInt(0);
                        String name = node.path("name").asText("");
                        if (id > 0) indexers.put(id, name.isBlank() ? "Indexer-" + id : name);
                    }
                } catch (Exception e) {
                    log.warn("Failed to parse indexer list: {}", e.getMessage());
                }
                if (!indexers.isEmpty()) cachedIndexers = indexers;
                log.info("Prowlarr indexers available: {}", indexers);
                return indexers;
            })
            .onErrorReturn(Map.of());
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
            Path cwd = Path.of("").toAbsolutePath();
            // Try configured path first, then fallbacks for different launch directories
            // (project root, backend/, backend/mp-spring/)
            List<Path> candidates = List.of(
                cwd.resolve(configXmlPath).normalize(),
                cwd.resolve("backend/go-server/prowlarr-data/config.xml"),
                cwd.resolve("go-server/prowlarr-data/config.xml"),
                cwd.resolve("../go-server/prowlarr-data/config.xml").normalize()
            );
            for (Path candidate : candidates) {
                try {
                    String xml = Files.readString(candidate);
                    Matcher m = API_KEY_RE.matcher(xml);
                    if (m.find()) {
                        resolvedApiKey = m.group(1);
                        lastFailedAt.set(0);
                        log.info("Prowlarr API key loaded from {}", candidate);
                        return resolvedApiKey;
                    }
                } catch (IOException ignored) {
                    // try next candidate
                }
            }
            log.warn("Cannot find prowlarr config.xml — CWD={}, tried: {}", cwd,
                candidates.stream().map(Path::toString).collect(java.util.stream.Collectors.joining(", ")));
            lastFailedAt.set(System.currentTimeMillis());
            return null;
        }
    }

    private Mono<List<SearchResult>> queryIndexerNewznab(String baseUrl, String key, int indexerId, String indexerName, String query) {
        String url = baseUrl + "/api/v1/indexer/" + indexerId + "/newznab?apikey=" + key
            + "&t=search&q=" + encode(query);
        return webClient.get()
            .uri(url)
            .retrieve()
            .bodyToMono(String.class)
            .map(xml -> parseTorznabXml(xml, indexerName))
            .onErrorResume(e -> {
                log.warn("Indexer {} ({}) search failed: {}", indexerId, indexerName, e.getMessage());
                return Mono.just(List.of());
            });
    }

    private List<SearchResult> parseTorznabXml(String xml, String indexerName) {
        List<SearchResult> results = new ArrayList<>();
        try {
            log.debug("Torznab XML ({} bytes) from {}", xml.length(), indexerName);
            DocumentBuilder builder = DB_FACTORY.newDocumentBuilder();
            Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));
            // indexerName comes from the Prowlarr API — always authoritative.
            // Channel <title> is ignored because Prowlarr always emits "Prowlarr" there.
            String channelIndexer = indexerName;

            NodeList items = doc.getElementsByTagName("item");
            for (int i = 0; i < items.getLength(); i++) {
                Element item    = (Element) items.item(i);
                String title    = text(item, "title");
                String pubDate  = text(item, "pubDate");
                String link     = text(item, "link");
                String guid     = text(item, "guid");
                long sizeBytes  = 0;
                int seeders = 0, peers = 0;
                String indexer  = channelIndexer;
                String magnetUrl = "";
                String infoHash  = "";

                // S1: <enclosure url="magnet:..." length="..."/>
                NodeList enclosures = item.getElementsByTagName("enclosure");
                if (enclosures.getLength() > 0) {
                    Element enc = (Element) enclosures.item(0);
                    String u = enc.getAttribute("url");
                    if (u.startsWith("magnet:")) magnetUrl = u;
                    String len = enc.getAttribute("length");
                    if (!len.isBlank()) sizeBytes = safeLong(len);
                }

                // S2: torznab:attr / newznab:attr — any namespace or none
                // getElementsByTagNameNS("*","attr") covers declared-namespace feeds;
                // fallback to getElementsByTagName("attr") covers no-namespace feeds.
                NodeList attrs = item.getElementsByTagNameNS("*", "attr");
                if (attrs.getLength() == 0) attrs = item.getElementsByTagName("attr");
                for (int j = 0; j < attrs.getLength(); j++) {
                    Element attr = (Element) attrs.item(j);
                    String name  = attr.getAttribute("name");
                    String val   = attr.getAttribute("value");
                    switch (name) {
                        case "seeders"   -> seeders   = safeInt(val);
                        case "peers"     -> peers     = safeInt(val);
                        // "leechers" is the alias used by several trackers
                        case "leechers"  -> { if (peers == 0) peers = safeInt(val); }
                        case "size"      -> { if (sizeBytes == 0) sizeBytes = safeLong(val); }
                        case "magneturl" -> { if (magnetUrl.isBlank()) magnetUrl = val; }
                        case "infohash"  -> infoHash = val;
                        case "indexer"   -> { /* ignored — authoritative name set from API */ }
                    }
                }

                // S3: <link>magnet:...
                if (magnetUrl.isBlank() && link.startsWith("magnet:")) magnetUrl = link;
                // S4: <guid isPermaLink="false">magnet:...
                if (magnetUrl.isBlank() && guid.startsWith("magnet:")) magnetUrl = guid;
                // S5: infohash attr → construct minimal magnet URI
                if (magnetUrl.isBlank() && !infoHash.isBlank()) {
                    magnetUrl = "magnet:?xt=urn:btih:" + infoHash;
                    if (!title.isBlank()) magnetUrl += "&dn=" + encode(title);
                }
                // S6: regex scan all text content — last resort for non-standard embeds
                if (magnetUrl.isBlank()) magnetUrl = scanMagnet(item.getTextContent());

                LinkedHashSet<String> tags = new LinkedHashSet<>();
                Matcher m = QUALITY.matcher(title);
                while (m.find()) tags.add(m.group(1).toUpperCase());
                results.add(new SearchResult(title, sizeBytes, seeders, peers, indexer, pubDate, magnetUrl, new ArrayList<>(tags)));
            }
        } catch (Exception e) {
            log.error("Torznab XML parse error: {}", e.getMessage());
        }
        return results;
    }

    private String scanMagnet(String text) {
        if (text == null || text.isBlank()) return "";
        Matcher m = MAGNET_RE.matcher(text);
        return m.find() ? m.group() : "";
    }

    private String text(Element parent, String tag) {
        NodeList nl = parent.getElementsByTagName(tag);
        return nl.getLength() == 0 ? "" : nl.item(0).getTextContent();
    }

    private int safeInt(String s) {
        try { return Integer.parseInt(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private long safeLong(String s) {
        try { return Long.parseLong(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private List<SearchResult> parseJsonResults(String json) {
        List<SearchResult> results = new ArrayList<>();
        try {
            JsonNode root = MAPPER.readTree(json);
            if (!root.isArray()) {
                log.error("Prowlarr search response not an array: {}", json.length() > 200 ? json.substring(0, 200) : json);
                return results;
            }
            for (JsonNode item : root) {
                String title     = item.path("title").asText("");
                long sizeBytes   = item.path("size").asLong(0);
                int seeders      = item.path("seeders").asInt(0);
                int peers        = item.path("peers").asInt(seeders);
                String indexer   = item.path("indexer").asText("");
                String pubDate   = item.path("publishDate").asText("");
                String magnetUrl = item.path("magnetUrl").asText("");
                if (magnetUrl.isBlank()) magnetUrl = item.path("downloadUrl").asText("");

                LinkedHashSet<String> tagSet = new LinkedHashSet<>();
                Matcher m = QUALITY.matcher(title);
                while (m.find()) tagSet.add(m.group(1).toUpperCase());

                results.add(new SearchResult(title, sizeBytes, seeders, peers, indexer, pubDate, magnetUrl, new ArrayList<>(tagSet)));
            }
        } catch (Exception e) {
            log.error("Prowlarr JSON parse error: {}", e.getMessage());
        }
        return results;
    }

    private String encode(String q) {
        return java.net.URLEncoder.encode(q, StandardCharsets.UTF_8);
    }
}
