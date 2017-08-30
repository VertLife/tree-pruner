#!/usr/bin/perl

use strict;
use DBI;
use List::Util 'shuffle';
use POSIX;

my $database = "birdtree";
my $username = "";
my $password = "";

my $pid  = shift @ARGV;
die "Need pid" if not ($pid =~ m/^[0-9]+$/) ;

my $treeset  = shift @ARGV;
die "Need treeset id" if not ($treeset =~ m/^[0-9]+$/) ;

my $treenum  = shift @ARGV;
die "Need number of trees" if not ($treenum =~ m/^[0-9]+$/) ;

my @goodNames = @ARGV;
die "Need more than two taxa" if ( $#goodNames < 2 ) ;

my $dbh = &ConnectToPg($database, $username, $password);

my $allTrees = $dbh->selectrow_array("SELECT count(*) FROM badtree tr WHERE tr.treeset_id = ? ", undef, $treeset);
die "Can't be fewer trees than requested" if ($treenum > $allTrees);

# create a random shuffle of all trees
my @shuffled_trees = shuffle( map { $_ + 1 } (0 .. $allTrees) );
# crop this list to the number requested by the user
@shuffled_trees = @shuffled_trees[0 .. ($treenum - 1)];

mkdir "/tmp/$pid";
chdir "/tmp/$pid";
open(OUTPUT, ">:utf8", "/tmp/$pid/sample") || die "Cannot open /tmp/$pid/sample!: $!";
	foreach my $gdnm (@goodNames) {
		$gdnm =~ s/ /_/g;
		print OUTPUT "taxon_1\t1\t$gdnm\n";
	}
close(OUTPUT);

open(TREE, ">:utf8", "/tmp/$pid.tre") || die "Cannot open /tmp/$pid/$pid.tre!: $!";
my $treeset_name = $dbh->selectrow_array ("SELECT treeset_name FROM treeset WHERE treeset_id = ?", undef, $treeset );
print TREE "#NEXUS\n\n[Tree distribution from: The global diversity of birds in space and time; W. Jetz, G. H. Thomas, J. B. Joy, K. Hartmann, A. O. Mooers doi:10.1038/nature11631]\n[Subsampled and pruned from birdtree.org on ".strftime("%m/%d/%Y %H:%M:%S", localtime)." ]\n[Data: \"$treeset_name\" (see Jetz et al. 2012 supplement for details)]\n\nBEGIN TREES;\n";
print "$treenum pruned trees selected at random from a pool of $allTrees trees:<br>\n";

my $statement = "SELECT thetree FROM badtree tr WHERE tr.tree_number = ? AND tr.treeset_id = ? LIMIT 1 ";
foreach my $cnt ( 0 .. $#shuffled_trees ) {

	open(OUTPUT, ">:utf8", "/tmp/$pid/phylo") || die "Cannot open /tmp/$pid/phylo!: $!";
		my $thetree = $dbh->selectrow_array ( $statement, undef, $shuffled_trees[$cnt], $treeset );
		print OUTPUT "$thetree";
	close(OUTPUT);

        

	print TREE "\tTREE tree_" . $shuffled_trees[$cnt] . " = ";
	my $pruned = `phylocom sampleprune`;
	if ($pruned =~ m/^\(\(.*\):0\.0+;$/) {
		$pruned =~ s/\):0\.0+;$/;/;
		$pruned =~ s/^\(//;
	}
	#This adds a set of parens if the root node has been crossed, paren ct should be Nspecies-1
	my $num_sp =()= $pruned =~ /_/g;
        my $num_op =()= $pruned =~ /\(/g;
	my $num_cp =()= $pruned =~ /\)/g;
        
        if(($num_sp-1)>$num_cp) {
		$pruned =~ s/;$/\);/;
		$pruned =~ s/^\(/\(\(/;			
	}
	
	print TREE "$pruned";
	
	print "Tree number $shuffled_trees[$cnt] done<br>\n";							
	
}
print TREE "END;\n\n";
close(TREE);


$dbh->disconnect;
exit;



#==============================================================
sub sqlclean {
	my ($input) = @_;
	
	# $input =~ s/(\%27)|(\')|(\-\-)|(\%23)|(#)//ix;
	$input =~ s/((\%3D)|(=))[^\n]*((\%27)|(\')|(\-\-)|(\%3B)|(;))//i;
	$input =~ s/\w*((\%27)|(\'))((\%6F)|o|(\%4F))((\%72)|r|(\%52))//ix;
	return($input);
}

# Connect to Postgres using DBI
#==============================================================
sub ConnectToPg {
 
    my ($cstr, $user, $pass) = @_;
  
    $cstr = "DBI:Pg:dbname="."$cstr";
    $cstr .= ";host=localhost";
  
    my $dbh = DBI->connect($cstr, $user, $pass, {PrintError => 1, RaiseError => 1});
    $dbh || &error("DBI connect failed : ",$dbh->errstr);
 
    return($dbh);
}

