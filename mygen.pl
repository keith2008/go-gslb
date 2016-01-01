#! /usr/bin/perl

use YAML::Syck;
use strict;

print Dump( \@ARGV );
print "@ARGV\n";

my $usage  = "$0 infile outfile FOO=REPLACEMENT_FOO BAR=REPLACEMENT_BAR ...";
my $source = shift @ARGV || die $usage;
my $dest   = shift @ARGV || die $usage;

my @replacements;

foreach (@ARGV) {
    die $usage unless ( $_ =~ /=/ );
    my ( $k, $v ) = split( /=/, $_, 2 );
    my $re = qr/\Q$k/;
    push( @replacements, [ $k, $v ] );
}

open( SOURCE, "<$source" )   || die "Failed to open $source : $!";
open( DEST,   ">$dest.new" ) || die "Failed to open $dest.new for write: $!";

while (<SOURCE>) {

    if ( $_ =~ m#//go:generate# ) {
        $_ =~ s#//go:generate#//\\\\//go:generated#;
        print DEST;
        next;
    }
    if ( $_ =~ m#//\+build generate# ) {
        print DEST "//AUTO GENERATED CODE\n";
        next;
    }

    while (1) {
        my $original = $_;
        foreach my $aref (@replacements) {
            my $ref = $aref->[0];
            my $val = $aref->[1];
            $_ =~ s/$ref/$val/g;
        }
        last if ( $original eq $_ );
    }

    print DEST;
}

close DEST;
close SOURCE;
rename( "$dest.new", "$dest" ) || die "Failed to rename $dest.new $dest: $!"
