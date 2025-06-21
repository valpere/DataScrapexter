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
my $no_enrich = 0;
my $help = 0;

# Parse command line arguments
GetOptions(
    'output|o=s' => \$output_file,
    'format|f=s' => \$output_format,
    'type|t=s' => \$input_type,
    'no-enrich' => \$no_enrich,
    'help|h' => \$help
) or die "Error in command line arguments\n";

# Show help if requested or no pattern provided
if ($help || @ARGV == 0) {
    show_usage();
    exit($help ? 0 : 1);
}

$pattern = $ARGV[0];

# Validate output format
unless ($output_format =~ /^(json|csv)$/) {
    die "Error: Invalid output format. Must be 'json' or 'csv'\n";
}

# Main execution
main();

sub main {
    # Process files
    combine_files($pattern, $input_type);
    
    # Enrich data if requested
    if (!$no_enrich && @combined_data) {
        enrich_data();
    }
    
    # Save results
    if (@combined_data) {
        save_combined_data($output_file, $output_format);
    } else {
        print "No data to save\n";
    }
    
    # Print summary
    print_summary();
}

sub show_usage {
    print <<'EOF';
Usage: combine_data.pl <pattern> [options]

Combine multiple DataScrapexter output files

Arguments:
  pattern            File pattern to match (e.g., "outputs/*.json")

Options:
  -o, --output FILE  Output file name (default: combined_data.json)
  -f, --format TYPE  Output format: json or csv (default: json)
  -t, --type TYPE    Input file type: json, csv, or auto (default: auto)
  --no-enrich        Skip data enrichment step
  -h, --help         Show this help message

Example:
  combine_data.pl "outputs/*.json" -o merged.json
  combine_data.pl "data/*.csv" -f csv -o combined.csv

EOF
}

sub combine_files {
    my ($file_pattern, $file_type) = @_;
    
    # Get list of files matching pattern
    my @files = glob($file_pattern);
    
    unless (@files) {
        print "No files found matching pattern: $file_pattern\n";
        return;
    }
    
    print "Found " . scalar(@files) . " files to process\n";
    
    foreach my $filepath (sort @files) {
        print "Processing: $filepath\n";
        
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
        print "  Added $added unique records (" . scalar(@records) . " total)\n";
    }
}

sub load_json_file {
    my ($filepath) = @_;
    my @records;
    
    eval {
        open my $fh, '<', $filepath or die "Cannot open file: $!\n";
        my $json_text = do { local $/; <$fh> };
        close $fh;
        
        my $data = decode_json($json_text);
        
        # Handle different JSON structures
        if (ref($data) eq 'ARRAY') {
            @records = @$data;
        } elsif ($format eq 'csv') {
        unless (@combined_data) {
            print "No data to save\n";
            return;
        }
        
        # Get all field names
        my %fieldnames;
        foreach my $record (@combined_data) {
            $fieldnames{$_} = 1 for keys %$record;
        }
        my @fieldnames = sort keys %fieldnames;
        
        # Create CSV writer
        my $csv = Text::CSV->new({ 
            binary => 1, 
            auto_diag => 1,
            eol => "\n"
        });
        
        open my $fh, '>:encoding(utf8)', $output_file or die "Cannot create output file: $!\n";
        
        # Write header
        $csv->print($fh, \@fieldnames);
        
        # Write data
        foreach my $record (@combined_data) {
            my @row;
            foreach my $field (@fieldnames) {
                push @row, $record->{$field} // '';
            }
            $csv->print($fh, \@row);
        }
        
        close $fh;
    }
    
    print "Saved " . scalar(@combined_data) . " records\n";
}

sub print_summary {
    print "\n" . "=" x 50 . "\n";
    print "Processing Summary\n";
    print "=" x 50 . "\n";
    print "Total files processed: " . scalar(keys %file_stats) . "\n";
    print "Total records processed: $total_processed\n";
    print "Unique records: " . scalar(@combined_data) . "\n";
    print "Duplicate records merged: $duplicates\n";
    
    if ($total_processed > 0) {
        printf "Deduplication rate: %.1f%%\n", ($duplicates / $total_processed * 100);
    }
    
    if (%file_stats) {
        print "\nRecords per file:\n";
        foreach my $filename (sort keys %file_stats) {
            print "  $filename: $file_stats{$filename}\n";
        }
    }
    
    # Field statistics
    if (@combined_data) {
        print "\nField coverage:\n";
        my %field_counts;
        
        foreach my $record (@combined_data) {
            foreach my $field (keys %$record) {
                if (defined $record->{$field} && $record->{$field} ne '') {
                    $field_counts{$field}++;
                }
            }
        }
        
        # Sort by count and show top 10
        my @sorted_fields = sort { $field_counts{$b} <=> $field_counts{$a} } keys %field_counts;
        my $max_fields = @sorted_fields > 10 ? 10 : @sorted_fields;
        
        for (my $i = 0; $i < $max_fields; $i++) {
            my $field = $sorted_fields[$i];
            my $count = $field_counts{$field};
            my $coverage = ($count / @combined_data) * 100;
            printf "  %s: %.1f%% (%d records)\n", $field, $coverage, $count;
        }
    }
} (ref($data) eq 'HASH') {
            # Check for common wrapper keys
            if (exists $data->{data}) {
                if (ref($data->{data}) eq 'ARRAY') {
                    @records = @{$data->{data}};
                } else {
                    @records = ($data->{data});
                }
            } elsif (exists $data->{results}) {
                if (ref($data->{results}) eq 'ARRAY') {
                    @records = @{$data->{results}};
                } else {
                    @records = ($data->{results});
                }
            } else {
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
            allow_loose_quotes => 1 
        });
        
        open my $fh, '<:encoding(utf8)', $filepath or die "Cannot open file: $!\n";
        
        # Read header
        my $header = $csv->getline($fh);
        $csv->column_names(@$header);
        
        # Read data
        while (my $row = $csv->getline_hr($fh)) {
            # Convert numeric strings to numbers
            foreach my $key (keys %$row) {
                my $value = $row->{$key};
                if (defined $value && $value ne '') {
                    # Try to convert to number
                    if ($value =~ /^-?\d+$/) {
                        $row->{$key} = int($value);
                    } elsif ($value =~ /^-?\d*\.?\d+$/) {
                        $row->{$key} = $value + 0;
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
    
    # Use key fields for deduplication
    my @key_fields = qw(url product_id id sku title name);
    my %hash_data;
    
    foreach my $field (@key_fields) {
        if (exists $record->{$field} && defined $record->{$field}) {
            $hash_data{$field} = $record->{$field};
        }
    }
    
    # If no key fields, use all data
    unless (%hash_data) {
        foreach my $key (keys %$record) {
            $hash_data{$key} = $record->{$key} if defined $record->{$key};
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
            elsif ($key =~ /^(timestamp|scraped_at|date)$/) {
                # Simple string comparison for ISO dates
                if ($value gt ($merged{$key} // '')) {
                    $merged{$key} = $value;
                }
            }
        }
    }
    
    return \%merged;
}

sub add_data {
    my ($records, $source_file) = @_;
    
    my $source_name = basename($source_file);
    my $added = 0;
    
    foreach my $record (@$records) {
        $total_processed++;
        
        # Add source file information
        $record->{_source_file} = $source_name;
        
        # Check for duplicates
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
    print "\nEnriching data...\n";
    
    foreach my $record (@combined_data) {
        # Add processing timestamp
        unless (exists $record->{_processed_at}) {
            $record->{_processed_at} = strftime("%Y-%m-%dT%H:%M:%S", localtime);
        }
        
        # Calculate price changes if historical data exists
        if (exists $record->{original_price} && exists $record->{current_price}) {
            my $original = $record->{original_price} || 0;
            my $current = $record->{current_price} || 0;
            
            if ($original > 0 && $original =~ /^[\d.]+$/ && $current =~ /^[\d.]+$/) {
                $record->{_price_change} = $current - $original;
                $record->{_price_change_percent} = (($current - $original) / $original) * 100;
            }
        }
        
        # Standardize boolean fields
        foreach my $bool_field (qw(in_stock available is_active)) {
            if (exists $record->{$bool_field}) {
                my $value = lc($record->{$bool_field} // '');
                $record->{$bool_field} = ($value =~ /^(true|yes|1|available|in stock)$/) ? 1 : 0;
            }
        }
    }
}

sub save_combined_data {
    my ($output_file, $format) = @_;
    
    print "\nSaving combined data to $output_file\n";
    
    if ($format eq 'json') {
        open my $fh, '>', $output_file or die "Cannot create output file: $!\n";
        print $fh JSON->new->pretty->encode(\@combined_data);
        close $fh;
        
    } elsif
