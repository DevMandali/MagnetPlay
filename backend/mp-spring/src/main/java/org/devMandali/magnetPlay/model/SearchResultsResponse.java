package org.devMandali.magnetPlay.model;

import java.util.List;

public record SearchResultsResponse(List<SearchResult> results, int total) {}
