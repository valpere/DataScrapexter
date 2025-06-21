#!/usr/bin/env perl
# Combine and process multiple DataScrapexter output files
# Supports merging JSON/CSV files with deduplication and enrichment

use strict;
use warnings;
use JSON;
use Text::CSV;
use File::Glob ':glob';
use File::Basename;
use Getopt::Long;
use Digest::MD5 qw(md5_hex);
use POSIX qw(strftime);
use Data::Dumper;

# Global variables
my @combined_data;
my %record_hashes;
my %file_stats;
my $total_processed = 0;
my $duplicates = 0;

# Command line options
my $pattern = '';
my $output_file = 'combined_data.json';
my $output_format = 'json';
my $input_type = 'auto';
my $merge_field = 'url';
my $hash_fields = '';
my $exclude_pattern = '';
my $no_enrich = 0;
my $dry_run = 0;
my $verbose = 0;
my $help = 0;

# Parse command line arguments
GetOptions(
    'sources=s' => \$pattern,
    'output|o=s' => \$output_file,
    'format|f=s' => \$output_format,
    'type|t=s' => \$input_type,
    'merge-field=s' => \$merge_field,
    'hash-fields=s' => \$hash_fields,
    'exclude=s' => \$exclude_pattern,
    'no-enrich' => \$no_enrich,
    'dry-run' => \$dry_run,
    'verbose|v' => \$verbose,
    'help|h' => \$help
) or die "Error in command line arguments\n";

# Show help if requested or no pattern provided
if ($help || (!$pattern && @ARGV == 0)) {
    show_usage();
    exit($help ? 0 : 1);
}

# Use positional argument if --sources not provided
$pattern ||= $ARGV[0];

unless ($pattern) {
    die "Error: No source files specified\n";
}

# Validate output format
unless ($output_format =~ /^(json|csv)$/) {
    die "Error: Invalid output format. Must be 'json' or 'csv'\n";
}

# Main execution
main();

sub main {
    print "=== DataScrapexter Data Combiner ===\n" if $verbose;
    print "Starting at: " . strftime("%Y-%m-%d %H:%M:%S", localtime) . "\n\n" if $verbose;
    
    # Process files
    combine_files($pattern, $input_type);
    
    # Enrich data if requested
    if (!$no_enrich && @combined_data) {
        print "\nEnriching data...\n" if $verbose;
        enrich_data();
    }
    
    # Save results
    if (@combined_data) {
        unless ($dry_run) {
            save_combined_data($output_file, $output_format);
        } else {
            print "\n[DRY RUN] Would save " . scalar(@combined_data) . " records to $output_file\n";
        }
    } else {
        print "No data to save\n";
    }
    
    # Print summary
    print_summary();
}

sub show_usage {
    print <<'EOF';
Usage: combine_data.pl --sources <pattern> [options]

Combine multiple DataScrapexter output files with deduplication

Required:
  --sources PATTERN   Source files or directories (supports wildcards)

Options:
  -o, --output FILE   Output file name (default: combined_data.json)
  -f, --format TYPE   Output format: json or csv (default: json)
  -t, --type TYPE     Input file type: json, csv, or auto (default: auto)
  --merge-field FIELD Field to use for matching records (default: url)
  --hash-fields LIST  Comma-separated fields for duplicate detection
  --exclude PATTERN   Pattern to exclude files
  --no-enrich         Skip data enrichment step
  --dry-run           Preview without creating output
  -v, --verbose       Enable verbose output
  -h, --help          Show this help message

Examples:
  combine_data.pl --sources "outputs/*.json" -o merged.json
  combine_data.pl --sources "data/*.csv" -f csv -o combined.csv
  combine_data.pl --sources "outputs/2024*" --exclude "*test*" --dry-run

EOF
}

sub combine_files {
    my ($file_pattern, $file_type) = @_;
    
    # Get list of files matching pattern
    my @files = glob($file_pattern);
    
    # Apply exclusion pattern if specified
    if ($exclude_pattern) {
        @files = grep { !/$exclude_pattern/ } @files;
    }
    
    unless (@files) {
        print "No files found matching pattern: $file_pattern\n";
        return;
    }
    
    print "Found " . scalar(@files) . " files to process\n";
    
    foreach my $filepath (sort @files) {
        next unless -f $filepath;  # Skip directories
        
        print "Processing: $filepath\n" if $verbose;
        
        # Determine file type
        my $current_type = $file_type;
        if ($file_type eq 'auto') {
            my ($name, $path, $ext) = fileparse($filepath, qr/\.[^.]*/);
            $current_type = ($ext eq '.json') ? 'json' : 'csv';
        }
        
        # Load data
        my @records;
        if ($current_type eq 'json') {
            @records = load_json_file($filepath);
        } else {
            @records = load_csv_file($filepath);
        }
        
        # Add to combined dataset
        my $added = add_data(\@records, $filepath);
        print "  Added $added unique records (" . scalar(@records) . " total)\n" if $verbose;
    }
}

sub load_json_file {
    my ($filepath) = @_;
    my @records;
    
    eval {
        open my $fh, '<:encoding(utf8)', $filepath or die "Cannot open file: $!\n";
        my $json_text = do { local $/; <$fh> };
        close $fh;
        
        my $data = decode_json($json_text);
        
        # Handle different JSON structures
        if (ref($data) eq 'ARRAY') {
            @records = @$data;
        } elsif (ref($data) eq 'HASH') {
            # Check for common wrapper keys
            if (exists $data->{data} && ref($data->{data}) eq 'ARRAY') {
                @records = @{$data->{data}};
            } elsif (exists $data->{results} && ref($data->{results}) eq 'ARRAY') {
                @records = @{$data->{results}};
            } elsif (exists $data->{items} && ref($data->{items}) eq 'ARRAY') {
                @records = @{$data->{items}};
            } else {
                # Single record
                @records = ($data);
            }
        }
    };
    
    if ($@) {
        warn "Error loading $filepath: $@\n";
    }
    
    return @records;
}

sub load_csv_file {
    my ($filepath) = @_;
    my @records;
    
    eval {
        my $csv = Text::CSV->new({ 
            binary => 1, 
            auto_diag => 1,
            allow_loose_quotes => 1,
            allow_whitespace => 1
        });
        
        open my $fh, '<:encoding(utf8)', $filepath or die "Cannot open file: $!\n";
        
        # Read header
        my $header = $csv->getline($fh);
        unless ($header && @$header) {
            die "Invalid CSV header\n";
        }
        
        # Clean header fields
        my @clean_header = map { s/^\s+|\s+$//g; $_ } @$header;
        $csv->column_names(@clean_header);
        
        # Read data
        while (my $row = $csv->getline_hr($fh)) {
            # Convert numeric strings to numbers
            foreach my $key (keys %$row) {
                my $value = $row->{$key};
                if (defined $value && $value ne '') {
                    # Remove leading/trailing whitespace
                    $value =~ s/^\s+|\s+$//g;
                    
                    # Try to convert to number
                    if ($value =~ /^-?\d+$/) {
                        $row->{$key} = int($value);
                    } elsif ($value =~ /^-?\d*\.?\d+$/) {
                        $row->{$key} = $value + 0;
                    } else {
                        $row->{$key} = $value;
                    }
                }
            }
            push @records, $row;
        }
        
        close $fh;
    };
    
    if ($@) {
        warn "Error loading $filepath: $@\n";
    }
    
    return @records;
}

sub generate_record_hash {
    my ($record) = @_;
    
    my @key_fields;
    
    # Use specified hash fields if provided
    if ($hash_fields) {
        @key_fields = split(/,/, $hash_fields);
    } else {
        # Default key fields for deduplication
        @key_fields = qw(url product_url link product_id id sku title name);
    }
    
    my %hash_data;
    
    foreach my $field (@key_fields) {
        $field =~ s/^\s+|\s+$//g;  # Trim whitespace
        if (exists $record->{$field} && defined $record->{$field} && $record->{$field} ne '') {
            $hash_data{$field} = $record->{$field};
        }
    }
    
    # If no key fields found, use all non-empty fields
    unless (%hash_data) {
        foreach my $key (keys %$record) {
            if (defined $record->{$key} && $record->{$key} ne '') {
                $hash_data{$key} = $record->{$key};
            }
        }
    }
    
    # Create hash
    my $json = JSON->new->canonical->encode(\%hash_data);
    return md5_hex($json);
}

sub merge_records {
    my ($existing, $new) = @_;
    
    my %merged = %$existing;
    
    foreach my $key (keys %$new) {
        my $value = $new->{$key};
        
        if (defined $value && $value ne '') {
            # Prefer non-empty values
            if (!exists $merged{$key} || !defined $merged{$key} || $merged{$key} eq '') {
                $merged{$key} = $value;
            }
            # For timestamps, keep the most recent
            elsif ($key =~ /^(timestamp|scraped_at|date|updated_at|created_at)$/i) {
                # Simple string comparison for ISO dates
                if ($value gt ($merged{$key} // '')) {
                    $merged{$key} = $value;
                }
            }
            # For prices, keep the most recent (assuming newer data)
            elsif ($key =~ /price/i && $value =~ /^[\d.]+$/) {
                $merged{$key} = $value;
            }
        }
    }
    
    # Update merge metadata
    $merged{_merge_count} = ($merged{_merge_count} // 1) + 1;
    $merged{_last_merged} = strftime("%Y-%m-%dT%H:%M:%S", localtime);
    
    return \%merged;
}

sub add_data {
    my ($records, $source_file) = @_;
    
    my $source_name = basename($source_file);
    my $added = 0;
    
    foreach my $record (@$records) {
        next unless ref($record) eq 'HASH';  # Skip invalid records
        
        $total_processed++;
        
        # Add source file information
        $record->{_source_file} = $source_name;
        
        # Generate hash for duplicate detection
        my $hash_val = generate_record_hash($record);
        
        if (exists $record_hashes{$hash_val}) {
            # Merge with existing record
            my $idx = $record_hashes{$hash_val};
            $combined_data[$idx] = merge_records($combined_data[$idx], $record);
            $duplicates++;
        } else {
            # Add new record
            push @combined_data, $record;
            $record_hashes{$hash_val} = $#combined_data;
            $added++;
        }
    }
    
    $file_stats{$source_name} = $added;
    return $added;
}

sub enrich_data {
    foreach my $record (@combined_data) {
        # Add processing timestamp if not exists
        unless (exists $record->{_processed_at}) {
            $record->{_processed_at} = strftime("%Y-%m-%dT%H:%M:%S", localtime);
        }
        
        # Calculate price changes if historical data exists
        if (exists $record->{original_price} && exists $record->{current_price}) {
            my $original = $record->{original_price} || 0;
            my $current = $record->{current_price} || 0;
            
            # Clean price values
            $original =~ s/[^\d.]//g if $original;
            $current =~ s/[^\d.]//g if $current;
            
            if ($original > 0 && $current > 0) {
                $record->{_price_change} = sprintf("%.2f", $current - $original);
                $record->{_price_change_percent} = sprintf("%.2f", (($current - $original) / $original) * 100);
            }
        }
        
        # Standardize boolean fields
        foreach my $bool_field (qw(in_stock available is_active on_sale)) {
            if (exists $record->{$bool_field}) {
                my $value = lc($record->{$bool_field} // '');
                $record->{$bool_field} = ($value =~ /^(true|yes|1|available|in stock|active|on)$/) ? 1 : 0;
            }
        }
        
        # Clean and standardize URLs
        foreach my $url_field (qw(url product_url link href)) {
            if (exists $record->{$url_field} && $record->{$url_field}) {
                # Remove trailing slashes and fragments
                $record->{$url_field} =~ s/\#.*$//;
                $record->{$url_field} =~ s/\/+$//;
            }
        }
    }
}

sub save_combined_data {
    my ($output_file, $format) = @_;
    
    print "\nSaving combined data to $output_file\n" if $verbose;
    
    if ($format eq 'json') {
        save_as_json($output_file);
    } elsif ($format eq 'csv') {
        save_as_csv($output_file);
    }
    
    print "Saved " . scalar(@combined_data) . " records\n";
}

sub save_as_json {
    my ($output_file) = @_;
    
    open my $fh, '>:encoding(utf8)', $output_file or die "Cannot create output file: $!\n";
    
    my $json = JSON->new->pretty->canonical;
    print $fh $json->encode(\@combined_data);
    
    close $fh;
}

sub save_as_csv {
    my ($output_file) = @_;
    
    unless (@combined_data) {
        print "No data to save\n";
        return;
    }
    
    # Get all field names
    my %fieldnames;
    foreach my $record (@combined_data) {
        $fieldnames{$_} = 1 for keys %$record;
    }
    
    # Sort fields, putting metadata fields at the end
    my @metadata_fields = grep { /^_/ } keys %fieldnames;
    my @data_fields = grep { !/^_/ } keys %fieldnames;
    my @sorted_fields = (sort @data_fields, sort @metadata_fields);
    
    # Create CSV writer
    my $csv = Text::CSV->new({ 
        binary => 1, 
        auto_diag => 1,
        eol => "\n"
    });
    
    open my $fh, '>:encoding(utf8)', $output_file or die "Cannot create output file: $!\n";
    
    # Write header
    $csv->print($fh, \@sorted_fields);
    
    # Write data
    foreach my $record (@combined_data) {
        my @row;
        foreach my $field (@sorted_fields) {
            my $value = $record->{$field} // '';
            # Convert references to JSON strings
            if (ref($value)) {
                $value = JSON->new->encode($value);
            }
            push @row, $value;
        }
        $csv->print($fh, \@row);
    }
    
    close $fh;
}

sub print_summary {
    print "\n" . "=" x 50 . "\n";
    print "Processing Summary\n";
    print "=" x 50 . "\n";
    print "Total files processed: " . scalar(keys %file_stats) . "\n";
    print "Total records processed: $total_processed\n";
    print "Unique records: " . scalar(@combined_data) . "\n";
    print "Duplicate records merged: $duplicates\n";
    
    if ($verbose && %file_stats) {
        print "\nRecords per file:\n";
        foreach my $file (sort keys %file_stats) {
            printf "  %-40s: %d records\n", $file, $file_stats{$file};
        }
    }
    
    # Show field coverage statistics
    if (@combined_data && $verbose) {
        print "\nField Coverage:\n";
        my %field_counts;
        
        foreach my $record (@combined_data) {
            foreach my $field (keys %$record) {
                next if $field =~ /^_/;  # Skip metadata fields
                $field_counts{$field}++ if defined $record->{$field} && $record->{$field} ne '';
            }
        }
        
        my @sorted_fields = sort { $field_counts{$b} <=> $field_counts{$a} } keys %field_counts;
        my $max_fields = @sorted_fields > 10 ? 10 : @sorted_fields;
        
        for (my $i = 0; $i < $max_fields; $i++) {
            my $field = $sorted_fields[$i];
            my $count = $field_counts{$field};
            my $coverage = ($count / @combined_data) * 100;
            printf "  %-30s: %.1f%% (%d records)\n", $field, $coverage, $count;
        }
    }
    
    print "=" x 50 . "\n";
}
