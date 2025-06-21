#!/usr/bin/env perl
# Price Analysis Script for DataScrapexter
# Analyzes price changes across scraping sessions and generates reports

use strict;
use warnings;
use JSON;
use File::Find;
use File::Basename;
use Getopt::Long;
use POSIX      qw(strftime);
use List::Util qw(sum max min);
use Data::Dumper;

# Configuration
my $data_dir    = '';
my $output_dir  = 'price_analysis';
my $days_back   = 30;
my $export_csv  = 0;
my $export_json = 0;

# Parse command line arguments
GetOptions(
    'data-dir=s'   => \$data_dir,
    'output-dir=s' => \$output_dir,
    'days=i'       => \$days_back,
    'csv'          => \$export_csv,
    'json'         => \$export_json,
    'help'         => sub {show_usage(); exit 0;}
) or die "Error in command line arguments\n";

# Validate arguments
unless ($data_dir) {
    print STDERR "Error: data directory is required\n";
    show_usage();
    exit 1;
}

unless (-d $data_dir) {
    die "Error: Data directory not found: $data_dir\n";
}

# Create output directory
unless (-d $output_dir) {
    mkdir $output_dir or die "Cannot create output directory: $!\n";
}

# Global price history storage
my %price_history;

# Main execution
main();

sub main {
    print "Loading data from last $days_back days...\n";

    # Load price data
    load_price_data();

    unless (keys %price_history) {
        print "No price data found to analyze\n";
        exit 0;
    }

    print "Analyzing price changes for " . scalar(keys %price_history) . " products...\n";

    # Analyze changes
    my $analysis = analyze_changes();

    # Generate report
    my $report = generate_report($analysis);
    print "\n$report\n";

    # Save report to file
    my $timestamp   = strftime("%Y%m%d_%H%M%S", localtime);
    my $report_file = "$output_dir/price_report_$timestamp.txt";

    open my $fh, '>', $report_file or die "Cannot create report file: $!\n";
    print $fh $report;
    close $fh;

    print "\nReport saved to: $report_file\n";

    # Export additional formats if requested
    if ($export_csv) {
        my $csv_file = "price_changes_$timestamp.csv";
        export_csv($analysis, $csv_file);
    }

    if ($export_json) {
        my $json_file = "price_alerts_$timestamp.json";
        generate_alerts_json($analysis, $json_file);
    }
} ## end sub main

sub show_usage {
    print <<'EOF';
Usage: analyze_price_changes.pl --data-dir <directory> [options]

Analyze price changes from DataScrapexter outputs

Required:
  --data-dir DIR     Directory containing scraped data files

Options:
  --output-dir DIR   Output directory for reports (default: price_analysis)
  --days N           Number of days to analyze (default: 30)
  --csv              Export detailed CSV report
  --json             Generate JSON alerts file
  --help             Show this help message

Example:
  analyze_price_changes.pl --data-dir outputs --days 7 --csv

EOF
} ## end sub show_usage

sub load_price_data {
    my $cutoff_time = time - ($days_back * 24 * 60 * 60);

    # Find all JSON files
    find(\&process_file, $data_dir);

    sub process_file {
        return unless /\.json$/;
        return unless -f $_;

        my $file_path = $File::Find::name;
        my $file_date = get_file_date($file_path);

        return if $file_date < $cutoff_time;

        # Load and process the file
        eval {
            open my $fh, '<', $file_path or die "Cannot open file: $!\n";
            my $json_text = do {local $/; <$fh>};
            close $fh;

            my $data = decode_json($json_text);
            process_data($data, $file_date, $file_path);
        };

        if ($@) {
            warn "Error loading $file_path: $@\n";
        }
    } ## end sub process_file
} ## end sub load_price_data

sub get_file_date {
    my ($file_path) = @_;

    # Try to extract date from filename (format: YYYYMMDD)
    if ($file_path =~ /(\d{8})/) {
        my $date_str = $1;
        my ($year, $month, $day) = ($date_str =~ /(\d{4})(\d{2})(\d{2})/);

        if ($year && $month && $day) {
            eval {
                use Time::Local;
                my $time = timelocal(0, 0, 0, $day, $month - 1, $year);
                return $time;
            };
        }
    }

    # Fall back to file modification time
    return (stat($file_path))[9];
} ## end sub get_file_date

sub process_data {
    my ($data, $date, $file_path) = @_;

    # Handle both array and single object
    my @items = ref($data) eq 'ARRAY' ? @$data : ($data);

    foreach my $item (@items) {
        # Handle nested data structure
        my $item_data = $item->{data} || $item;

        # Extract product identifier and price
        my $product_id = get_product_id($item_data);
        my $price      = extract_price($item_data);

        if ($product_id && defined $price) {
            $price_history{$product_id} ||= {
                name   => $item_data->{title} || $item_data->{product_name} || $product_id,
                prices => []
            };

            push @{$price_history{$product_id}{prices}}, {
                date  => $date,
                price => $price,
                url   => $item->{url} || ''
            };
        }
    } ## end foreach my $item (@items)
} ## end sub process_data

sub get_product_id {
    my ($data) = @_;

    # Try different field names
    foreach my $field (qw(product_id sku id product_url url)) {
        if (exists $data->{$field} && $data->{$field}) {
            return $data->{$field};
        }
    }

    # Generate ID from title if available
    if ($data->{title} || $data->{product_name}) {
        my $title = $data->{title} || $data->{product_name};
        $title = lc($title);
        $title =~ s/\s+/_/g;
        return substr($title, 0, 50);
    }

    return undef;
} ## end sub get_product_id

sub extract_price {
    my ($data) = @_;

    # Try different price field names
    foreach my $field (qw(price current_price sale_price regular_price)) {
        if (exists $data->{$field}) {
            my $value = $data->{$field};

            # Handle different price formats
            if ($value =~ /^[\d.]+$/) {
                return $value +0;    # Convert to number
            }
            elsif ($value =~ /([\d,]+\.?\d*)/) {
                my $price_str = $1;
                $price_str =~ s/,//g;
                return $price_str +0;
            }
        }
    }

    return undef;
} ## end sub extract_price

sub analyze_changes {
    my %analysis = (
        total_products         => scalar(keys %price_history),
        products_with_changes  => 0,
        average_change_percent => 0,
        biggest_increases      => [],
        biggest_decreases      => [],
        price_alerts           => [],
        summary_stats          => {}
    );

    my @all_changes;

    foreach my $product_id (keys %price_history) {
        my $product_data = $price_history{$product_id};
        my @prices       = sort {$a->{date} <=> $b->{date}} @{$product_data->{prices}};

        next if @prices < 2;

        # Calculate price change
        my $old_price = $prices[0]->{price};
        my $new_price = $prices[-1]->{price};

        next if $old_price == 0;

        my $change_amount  = $new_price - $old_price;
        my $change_percent = ($change_amount / $old_price) * 100;

        if ($change_amount != 0) {
            $analysis{products_with_changes}++;
            push @all_changes, $change_percent;

            my $change_data = {
                product        => $product_data->{name},
                product_id     => $product_id,
                old_price      => $old_price,
                new_price      => $new_price,
                change_amount  => $change_amount,
                change_percent => $change_percent,
                first_seen     => strftime("%Y-%m-%d", localtime($prices[0]->{date})),
                last_seen      => strftime("%Y-%m-%d", localtime($prices[-1]->{date})),
                url            => $prices[-1]->{url}
            };

            # Track significant changes
            if ($change_percent > 5) {
                push @{$analysis{biggest_increases}}, $change_data;
            }
            elsif ($change_percent < -5) {
                push @{$analysis{biggest_decreases}}, $change_data;
            }

            # Generate alerts for large changes
            if (abs($change_percent) > 10) {
                push @{$analysis{price_alerts}}, $change_data;
            }
        } ## end if ($change_amount != ...)
    } ## end foreach my $product_id (keys...)

    # Calculate summary statistics
    if (@all_changes) {
        $analysis{average_change_percent} = sum(@all_changes) / @all_changes;
        $analysis{summary_stats}          = {
            mean_change   => sum(@all_changes) / @all_changes,
            median_change => median(@all_changes),
            std_dev       => std_dev(@all_changes),
            min_change    => min(@all_changes),
            max_change    => max(@all_changes)
        };
    }

    # Sort by change magnitude
    @{$analysis{biggest_increases}} = sort {$b->{change_percent} <=> $a->{change_percent}} @{$analysis{biggest_increases}};
    @{$analysis{biggest_decreases}} = sort {$a->{change_percent} <=> $b->{change_percent}} @{$analysis{biggest_decreases}};

    return \%analysis;
} ## end sub analyze_changes

sub generate_report {
    my ($analysis) = @_;

    my @report;
    push @report, "=" x 60;
    push @report, "Price Analysis Report - " . strftime("%Y-%m-%d %H:%M", localtime);
    push @report, "=" x 60;
    push @report, "";

    # Summary
    push @report, "SUMMARY";
    push @report, "-" x 20;
    push @report, "Total products tracked: $analysis->{total_products}";
    push @report, "Products with price changes: $analysis->{products_with_changes}";

    if ($analysis->{average_change_percent} != 0) {
        push @report, sprintf("Average price change: %.2f%%", $analysis->{average_change_percent});
    }

    if ($analysis->{summary_stats} && %{$analysis->{summary_stats}}) {
        my $stats = $analysis->{summary_stats};
        push @report, sprintf("Median change: %.2f%%", $stats->{median_change});
        push @report, sprintf(
            "Price change range: %.2f%% to %.2f%%",
            $stats->{min_change}, $stats->{max_change}
        );
    }

    push @report, "";

    # Price increases
    if (@{$analysis->{biggest_increases}}) {
        push @report, "TOP PRICE INCREASES";
        push @report, "-" x 20;

        my $count = 0;
        foreach my $item (@{$analysis->{biggest_increases}}) {
            last if ++$count > 10;

            push @report, substr($item->{product}, 0, 50);
            push @report, sprintf(
                "  \$%.2f → \$%.2f (%+.1f%%)",
                $item->{old_price}, $item->{new_price}, $item->{change_percent}
            );
            push @report, "  Period: $item->{first_seen} to $item->{last_seen}";
            push @report, "";
        }
    }

    # Price decreases
    if (@{$analysis->{biggest_decreases}}) {
        push @report, "TOP PRICE DECREASES";
        push @report, "-" x 20;

        my $count = 0;
        foreach my $item (@{$analysis->{biggest_decreases}}) {
            last if ++$count > 10;

            push @report, substr($item->{product}, 0, 50);
            push @report, sprintf(
                "  \$%.2f → \$%.2f (%+.1f%%)",
                $item->{old_price}, $item->{new_price}, $item->{change_percent}
            );
            push @report, "  Period: $item->{first_seen} to $item->{last_seen}";
            push @report, "";
        }
    }

    # Alerts
    if (@{$analysis->{price_alerts}}) {
        push @report, "PRICE ALERTS (>10% change)";
        push @report, "-" x 20;

        foreach my $item (@{$analysis->{price_alerts}}) {
            push @report, "⚠️  " . substr($item->{product}, 0, 50);
            push @report, sprintf(
                "   %+.1f%% change (\$%+.2f)",
                $item->{change_percent}, $item->{change_amount}
            );
            push @report, "";
        }
    }

    return join("\n", @report);
} ## end sub generate_report

sub export_csv {
    my ($analysis, $filename) = @_;

    my $csv_path = "$output_dir/$filename";
    open my $fh, '>', $csv_path or die "Cannot create CSV file: $!\n";

    # Write header
    print $fh "Product,Product ID,Old Price,New Price,Change Amount,Change Percent,";
    print $fh "First Seen,Last Seen,Price Points,URL\n";

    # Collect all products with changes
    my @all_changes;

    foreach my $product_id (keys %price_history) {
        my $product_data = $price_history{$product_id};
        my @prices       = sort {$a->{date} <=> $b->{date}} @{$product_data->{prices}};

        next if @prices < 2;

        my $old_price = $prices[0]->{price};
        my $new_price = $prices[-1]->{price};

        next if $old_price == 0;

        my $change_amount  = $new_price - $old_price;
        my $change_percent = ($change_amount / $old_price) * 100;

        push @all_changes, {
            product        => $product_data->{name},
            product_id     => $product_id,
            old_price      => $old_price,
            new_price      => $new_price,
            change_amount  => $change_amount,
            change_percent => $change_percent,
            first_seen     => strftime("%Y-%m-%d", localtime($prices[0]->{date})),
            last_seen      => strftime("%Y-%m-%d", localtime($prices[-1]->{date})),
            price_points   => scalar(@prices),
            url            => $prices[-1]->{url}
        };
    } ## end foreach my $product_id (keys...)

    # Sort by change percentage
    @all_changes = sort {$b->{change_percent} <=> $a->{change_percent}} @all_changes;

    # Write data
    foreach my $row (@all_changes) {
        print $fh qq("$row->{product}","$row->{product_id}",);
        print $fh sprintf(
            '$%.2f,$%.2f,$%.2f,%.1f%%,',
            $row->{old_price},     $row->{new_price},
            $row->{change_amount}, $row->{change_percent}
        );
        print $fh qq("$row->{first_seen}","$row->{last_seen}",);
        print $fh qq($row->{price_points},"$row->{url}"\n);
    }

    close $fh;
    print "Exported " . scalar(@all_changes) . " price changes to $csv_path\n";
} ## end sub export_csv

sub generate_alerts_json {
    my ($analysis, $filename) = @_;

    my $alerts_path = "$output_dir/$filename";

    my $alerts_data = {
        generated => strftime("%Y-%m-%dT%H:%M:%S", localtime),
        summary   => {
            total_products   => $analysis->{total_products},
            products_changed => $analysis->{products_with_changes},
            average_change   => $analysis->{average_change_percent}
        },
        alerts    => $analysis->{price_alerts},
        increases => [@{$analysis->{biggest_increases}}[0 .. 4]],
        decreases => [@{$analysis->{biggest_decreases}}[0 .. 4]]
    };

    open my $fh, '>', $alerts_path or die "Cannot create alerts file: $!\n";
    print $fh encode_json($alerts_data);
    close $fh;

    print "Generated alerts file: $alerts_path\n";
} ## end sub generate_alerts_json

# Utility functions
sub median {
    my @values = sort {$a <=> $b} @_;
    my $count  = @values;

    return 0 unless $count;

    if ($count % 2) {
        return $values[int($count / 2)];
    }
    else {
        return ($values[$count / 2 - 1] + $values[$count / 2]) / 2;
    }
}

sub std_dev {
    my @values = @_;
    my $count  = @values;

    return 0 if $count < 2;

    my $mean   = sum(@values) / $count;
    my $sum_sq = sum(map {($_ - $mean)**2} @values);

    return sqrt($sum_sq / ($count - 1));
}
