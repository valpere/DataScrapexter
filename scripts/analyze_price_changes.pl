#!/usr/bin/env perl
# Script to identify and scrape new products not yet in the database
# Compares current listings with historical data to find new items

use strict;
use warnings;
use JSON;
use File::Find;
use File::Basename;
use Getopt::Long;
use POSIX qw(strftime);
use Digest::MD5 qw(md5_hex);

# Configuration
my $listings_file = '';
my $history_dir = '';
my $config_file = '';
my $output_dir = 'outputs/new_products';
my $max_products = 50;
my $delay = 3;
my $verbose = 0;

# Parse command line arguments
GetOptions(
    'listings=s' => \$listings_file,
    'history=s' => \$history_dir,
    'config=s' => \$config_file,
    'output=s' => \$output_dir,
    'max=i' => \$max_products,
    'delay=i' => \$delay,
    'verbose' => \$verbose,
    'help' => sub { show_usage(); exit 0; }
) or die "Error in command line arguments\n";

# Validate required arguments
unless ($listings_file && $history_dir && $config_file) {
    print STDERR "Error: Missing required arguments\n";
    show_usage();
    exit 1;
}

# Validate files exist
die "Listings file not found: $listings_file\n" unless -f $listings_file;
die "History directory not found: $history_dir\n" unless -d $history_dir;
die "Config file not found: $config_file\n" unless -f $config_file;

# Create output directory
unless (-d $output_dir) {
    mkdir $output_dir or die "Cannot create output directory: $!\n";
}

# Global data storage
my %known_products;
my @new_products;
my $timestamp = strftime("%Y%m%d_%H%M%S", localtime);

# Main execution
main();

sub main {
    print "=== New Product Discovery and Scraping ===\n";
    print "Timestamp: " . strftime("%Y-%m-%d %H:%M:%S", localtime) . "\n\n";
    
    # Load historical product data
    print "Loading historical product data...\n";
    load_historical_data();
    print "Found " . scalar(keys %known_products) . " known products\n\n";
    
    # Load current listings
    print "Loading current listings...\n";
    my @current_products = load_current_listings();
    print "Found " . scalar(@current_products) . " current products\n\n";
    
    # Identify new products
    print "Identifying new products...\n";
    identify_new_products(\@current_products);
    print "Found " . scalar(@new_products) . " new products\n\n";
    
    if (@new_products) {
        # Limit number of products to scrape
        if (@new_products > $max_products) {
            @new_products = @new_products[0..$max_products-1];
            print "Limiting to first $max_products products\n\n";
        }
        
        # Create URL file for scraping
        my $url_file = "$output_dir/new_product_urls_$timestamp.txt";
        save_urls($url_file);
        
        # Run scraper on new products
        print "Scraping new products...\n";
        scrape_new_products($url_file);
        
        # Generate report
        generate_report();
    } else {
        print "No new products to scrape\n";
    }
}

sub show_usage {
    print <<'EOF';
Usage: scrape_new_products.pl --listings <file> --history <dir> --config <file> [options]

Identify and scrape new products not in historical data

Required:
  --listings FILE    Current product listings file (JSON/CSV)
  --history DIR      Directory containing historical product data
  --config FILE      DataScrapexter configuration file for product details

Options:
  --output DIR       Output directory (default: outputs/new_products)
  --max N            Maximum number of new products to scrape (default: 50)
  --delay N          Delay between products in seconds (default: 3)
  --verbose          Enable verbose output
  --help             Show this help message

Example:
  scrape_new_products.pl \
    --listings outputs/products-listing.csv \
    --history outputs/products \
    --config configs/product-details.yaml \
    --max 20

EOF
}

sub load_historical_data {
    # Find all JSON files in history directory
    find(\&process_history_file, $history_dir);
    
    sub process_history_file {
        return unless /\.json$/;
        return unless -f $_;
        
        my $file_path = $File::Find::name;
        
        eval {
            open my $fh, '<', $file_path or die "Cannot open file: $!\n";
            my $json_text = do { local $/; <$fh> };
            close $fh;
            
            my $data = decode_json($json_text);
            
            # Handle different JSON structures
            my @items = ref($data) eq 'ARRAY' ? @$data : ($data);
            
            foreach my $item (@items) {
                my $product_data = $item->{data} || $item;
                
                # Extract product identifier
                my $product_id = extract_product_id($product_data);
                if ($product_id) {
                    $known_products{$product_id} = {
                        first_seen => $product_data->{_scraped_at} || 
                                     $product_data->{timestamp} || 
                                     strftime("%Y-%m-%d", localtime((stat($file_path))[9])),
                        file => basename($file_path)
                    };
                }
            }
        };
        
        if ($@ && $verbose) {
            warn "Error loading $file_path: $@\n";
        }
    }
}

sub load_current_listings {
    my @products;
    
    # Detect file type
    my $ext = (fileparse($listings_file, qr/\.[^.]*/))[2];
    
    if ($ext eq '.json') {
        # Load JSON file
        open my $fh, '<', $listings_file or die "Cannot open listings file: $!\n";
        my $json_text = do { local $/; <$fh> };
        close $fh;
        
        my $data = decode_json($json_text);
        @products = ref($data) eq 'ARRAY' ? @$data : ($data);
        
    } elsif ($ext eq '.csv') {
        # Load CSV file
        require Text::CSV;
        my $csv = Text::CSV->new({ binary => 1, auto_diag => 1 });
        
        open my $fh, '<:encoding(utf8)', $listings_file or die "Cannot open listings file: $!\n";
        
        # Read header
        my $header = $csv->getline($fh);
        $csv->column_names(@$header);
        
        # Read data
        while (my $row = $csv->getline_hr($fh)) {
            push @products, $row;
        }
        
        close $fh;
    } else {
        die "Unsupported file format: $ext\n";
    }
    
    return @products;
}

sub extract_product_id {
    my ($product) = @_;
    
    # Try different field names for product ID
    foreach my $field (qw(product_url url product_id id sku)) {
        if (exists $product->{$field} && defined $product->{$field} && $product->{$field} ne '') {
            return $product->{$field};
        }
    }
    
    # Generate ID from title if available
    if ($product->{title} || $product->{product_name} || $product->{name}) {
        my $title = $product->{title} || $product->{product_name} || $product->{name};
        return generate_id_from_title($title);
    }
    
    return undef;
}

sub generate_id_from_title {
    my ($title) = @_;
    
    # Create a normalized ID from title
    my $id = lc($title);
    $id =~ s/[^a-z0-9]+/_/g;
    $id =~ s/^_|_$//g;
    
    # Add hash for uniqueness
    my $hash = substr(md5_hex($title), 0, 8);
    
    return "${id}_${hash}";
}

sub identify_new_products {
    my ($current_products) = @_;
    
    foreach my $product (@$current_products) {
        # Handle nested data
        my $product_data = $product->{data} || $product;
        
        # Extract product ID
        my $product_id = extract_product_id($product_data);
        
        if ($product_id && !exists $known_products{$product_id}) {
            # This is a new product
            push @new_products, {
                id => $product_id,
                url => $product_data->{product_url} || $product_data->{url},
                name => $product_data->{product_name} || 
                        $product_data->{title} || 
                        $product_data->{name} || 
                        'Unknown',
                price => $product_data->{price} || $product_data->{current_price},
                data => $product_data
            };
            
            print "  New: $new_products[-1]->{name}\n" if $verbose;
        }
    }
}

sub save_urls {
    my ($url_file) = @_;
    
    open my $fh, '>', $url_file or die "Cannot create URL file: $!\n";
    
    foreach my $product (@new_products) {
        if ($product->{url}) {
            print $fh "$product->{url}\n";
        }
    }
    
    close $fh;
    
    print "Saved " . scalar(@new_products) . " URLs to $url_file\n\n";
}

sub scrape_new_products {
    my ($url_file) = @_;
    
    # Check if datascrapexter is available
    my $datascrapexter = `which datascrapexter 2>/dev/null`;
    chomp $datascrapexter;
    
    unless ($datascrapexter) {
        die "datascrapexter not found in PATH\n";
    }
    
    # Validate configuration
    system("datascrapexter validate '$config_file' >/dev/null 2>&1");
    if ($? != 0) {
        die "Invalid configuration file: $config_file\n";
    }
    
    my $successful = 0;
    my $failed = 0;
    
    foreach my $product (@new_products) {
        next unless $product->{url};
        
        print "Scraping: $product->{name}\n";
        print "  URL: $product->{url}\n" if $verbose;
        
        # Generate output filename
        my $safe_name = $product->{id};
        $safe_name =~ s/[^a-zA-Z0-9-_]/_/g;
        my $output_file = "$output_dir/${safe_name}_$timestamp.json";
        
        # Set environment variable for URL
        $ENV{PRODUCT_URL} = $product->{url};
        
        # Run scraper
        my $cmd = "datascrapexter run '$config_file' -o '$output_file' 2>&1";
        my $output = `$cmd`;
        my $exit_code = $? >> 8;
        
        if ($exit_code == 0 && -f $output_file && -s $output_file) {
            $successful++;
            print "  ✓ Success\n";
            
            # Add metadata to scraped file
            add_metadata($output_file, $product);
        } else {
            $failed++;
            print "  ✗ Failed\n";
            print "  Error: $output\n" if $verbose && $output;
            unlink $output_file if -f $output_file;
        }
        
        # Rate limiting
        sleep $delay if $delay > 0;
    }
    
    print "\nScraping complete: $successful successful, $failed failed\n";
}

sub add_metadata {
    my ($file, $product) = @_;
    
    # Read scraped data
    open my $fh, '<', $file or return;
    my $json_text = do { local $/; <$fh> };
    close $fh;
    
    my $data = decode_json($json_text);
    
    # Add metadata
    if (ref($data) eq 'ARRAY' && @$data > 0) {
        $data->[0]->{_new_product} = JSON::true;
        $data->[0]->{_discovered_at} = strftime("%Y-%m-%dT%H:%M:%S", localtime);
        $data->[0]->{_listing_price} = $product->{price} if $product->{price};
    }
    
    # Write back
    open $fh, '>', $file or return;
    print $fh encode_json($data);
    close $fh;
}

sub generate_report {
    my $report_file = "$output_dir/new_products_report_$timestamp.txt";
    
    open my $fh, '>', $report_file or die "Cannot create report file: $!\n";
    
    print $fh "New Products Discovery Report\n";
    print $fh "=" x 50 . "\n";
    print $fh "Generated: " . strftime("%Y-%m-%d %H:%M:%S", localtime) . "\n\n";
    
    print $fh "Summary:\n";
    print $fh "- Known products in database: " . scalar(keys %known_products) . "\n";
    print $fh "- New products discovered: " . scalar(@new_products) . "\n";
    print $fh "- Products scraped: " . ($max_products < @new_products ? $max_products : scalar(@new_products)) . "\n\n";
    
    print $fh "New Products List:\n";
    print $fh "-" x 50 . "\n";
    
    foreach my $product (@new_products) {
        print $fh "\nProduct: $product->{name}\n";
        print $fh "ID: $product->{id}\n";
        print $fh "URL: $product->{url}\n" if $product->{url};
        print $fh "Price: $product->{price}\n" if $product->{price};
    }
    
    close $fh;
    
    print "\nReport saved to: $report_file\n";
}
