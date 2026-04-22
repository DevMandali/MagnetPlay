package org.devMandali.magnetPlay.model;

import java.util.List;

public record SearchResult(
    String title,
    long sizeBytes,
    int seeders,
    int peers,
    String indexer,
    String pubDate,
    String magnetUrl,
    List<String> qualityTags
) {}
