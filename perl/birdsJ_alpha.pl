#!/usr/bin/perl

use strict;
use DBI;
use POSIX;
#use List::Util 'shuffle';
use JSON;
do 'valnames.pl';


my $database = "birdtree";
my $username = "";
my $password = "";

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);
my $dbh = &ConnectToPg($database, $username, $password);
my $pid = $$;
my %json = ();

print "Access-Control-Allow-Origin: *\n";
print "Content-type: application/json\n\n";

dumpData($dbh, $pid);
print encode_json(\%json);

$dbh->disconnect;
exit(0);


# Declare the subroutines
sub trim($);
sub ltrim($);
sub rtrim($);

# Perl trim function to remove whitespace from the start and end of the string
sub trim($)
{
	my $string = shift;
	$string =~ s/^\s+//;
	$string =~ s/\s+$//;
	return $string;
}
# Left trim function to remove leading whitespace
sub ltrim($)
{
	my $string = shift;
	$string =~ s/^\s+//;
	return $string;
}
# Right trim function to remove trailing whitespace
sub rtrim($)
{
	my $string = shift;
	$string =~ s/\s+$//;
	return $string;
}

#==============================================================
sub dumpData {
	
	my ($dbh, $pid) = @_;
 
	if ($frmVals{'email'} =~ m/^([^@]+\@[^@]+)$/) {
		my $email = &sqlclean($1);
		
		if ($frmVals{'treeset'} =~ m/^([0-9]+)$/) {
			my $treeset = $1;
			
			if ($frmVals{'treenum'} =~ m/^([0-9]+)$/) {
				my $treenum = $1;

				my $species = &sqlclean( $frmVals{'species'} );
				my @inNames = split(/\n/, $species );
				
				my @goodNames;
				my @badNames;
				my $statement = 'SELECT count(*) FROM taxon_treeset ttr JOIN taxon tx USING (taxon_id) ';
				$statement .= 'WHERE ttr.treeset_id = ? AND tx.taxon_name = ? ';
				foreach my $in ( @inNames ) {
					$in =~ s/\f//g;
					$in =~ s/\r//g;
					$in = trim($in);
					my $totRec = $dbh->selectrow_array($statement, undef, $treeset, $in);
					if ($totRec == 1) {
						push (@goodNames, $in);
					} else {
						push (@badNames, $in);
					}
				}
		
				if ( $badNames[0] ) {
					open(BADNAMES, ">:utf8", "/tmp/$pid.invalid_names.txt") ||  die "Cannot open /tmp/$pid.invalid_names.txt!: $!";
					foreach my $binomial ( @badNames ) {
						print BADNAMES "$binomial\n"; 
						push @{ $json{results} }, { "bad_name" => $binomial } ;
					}
					close(BADNAMES);
				}
				
				if ( scalar(@goodNames) > 2 ) {
					if((scalar(@goodNames) < 4501) or ((scalar(@goodNames) < 4501) and ($frmVals{'debug'} eq 'tinamus'))) {
						$dbh->do( "INSERT INTO usage (email, treeset_id, number_of_trees, process) VALUES ( ?, ?, ?, ? )", undef, $email, $treeset, $treenum, $pid );
						my $allTrees = $dbh->selectrow_array("SELECT count(*) FROM tree tr WHERE tr.treeset_id = ? ", undef, $treeset);
						if ($treenum <= $allTrees) {
					
							#my $accessions = $dbh->prepare ("SELECT accession, gene, citation FROM accession_numbers WHERE binomial = ? ORDER BY gene");
							#open(CITATIONS, ">:utf8", "/tmp/$pid.txt") || die "Cannot open /tmp/$pid.txt!: $!";
							#foreach my $binomial ( @goodNames ) {
							#	my $totRec = $dbh->selectrow_array ("SELECT COUNT(*) FROM accession_numbers WHERE binomial = ?", undef, $binomial);
							#	if ($totRec) {
							#		$accessions->execute($binomial);
							#		for my $row (@{$accessions->fetchall_arrayref}) {
							#			print CITATIONS "$binomial\t" . join( "\t", @$row ) . "\n";
							#		}
							#		$accessions->finish;						
							#	}
							#}
							#close(CITATIONS);
							my $isbeta = '';
							#if($frmVals{'beta'} eq 'true') {
							#	my $cmd = "perl nohupbirds_beta.pl $pid $treeset $treenum '";
							#} else {
								my $cmd = "perl nohupbirds_alpha.pl $pid $treeset $treenum '";
							#}							
							$cmd .= join("' '", @goodNames);
							$cmd .= "'  >> /tmp/$pid.log" . ' 2>&1 &';
					
							open(LOGGER, ">:utf8", "/tmp/$pid.cmd.log") ||  die "Cannot open /tmp/$pid.cmd.log!: $!";
							print LOGGER "$cmd";
							close(LOGGER);
	
							#if ( ! fork() ) {
								system("$cmd");
							#}

							push @{ $json{results} }, { "trees_url" => "/bird-tree/cgi-bin/birdme.pl?file=$pid.tre", "citations_url" => "/bird-tree/cgi-bin/birdme.pl?file=$pid.txt", "pid" => $pid, "trees" => $treenum } ;
						} else {
							push @{ $json{results} }, { "error_message" => "Please request a subsample of trees that is between 1 and $allTrees" } ;
						}
					} else {
						if($frmVals{'debug'} ne 'tinamus') {
							push @{ $json{results} }, { "error_message" => "This tool is in development and currently limited to 4500 species." };						
						} else {
							push @{ $json{results} }, { "error_message" => "Please limit your selection to 4500 species or less." } ;
							
						}
					}
				} else {
					push @{ $json{results} }, { "error_message" => "Please provide a greater number of valid taxa" } ;
				}
			} else {
				push @{ $json{results} }, { "error_message" => "Please provide a valid number of trees" } ;
			}
		} else {
			push @{ $json{results} }, { "error_message" => "Please provide a valid treeset" } ;
		}
	} else {
		push @{ $json{results} }, { "error_message" => "Please provide an email address" } ;
	}
}

#==============================================================
sub sqlclean {
	my ($input) = @_;
	
	# $input =~ s/(\%27)|(\')|(\-\-)|(\%23)|(#)//ix;
	$input =~ s/((\%3D)|(=))[^\n]*((\%27)|(\')|(\-\-)|(\%3B)|(;))//i;
	$input =~ s/\w*((\%27)|(\'))((\%6F)|o|(\%4F))((\%72)|r|(\%52))//ix;
	return($input);
}
	
# provide a popup list of the available datasets
#==============================================================
sub listTreesets {
	my ($dbh) = @_;

	my $statement = 'SELECT ts.treeset_id, ts.treeset_name, ';
	$statement .= '(SELECT count(*) FROM tree tr WHERE tr.treeset_id = ts.treeset_id) AS "Number of Trees", ';
	$statement .= '(SELECT count(*) FROM taxon_treeset tt WHERE tt.treeset_id = ts.treeset_id ) AS "Number of Taxa" ';
	$statement .= 'FROM treeset ts ORDER BY ts.treeset_name ';
	my $sth = $dbh->prepare( $statement ) or die "Can't prepare $statement: $dbh->errstr\n";		
	my $rv = $sth->execute or die "can't execute the query: $sth->errstr\n";

	my @values;
	my %labels;
	 
	print '<select name="treeset" >';
	while(my @row = $sth->fetchrow_array) {
		print '<option value="' . $row[0] . '">';
		print $row[1] . ": a set of " . $row[2] . " trees with " . $row[3] . " OTUs each ";
		print '</option>';
	}
	print '</select>';
	my $rd = $sth->finish;

}

#==============================================================
sub ReadForm {
	my ($varRef) = @_;
	my ($i, $key, $val, $variable, @list);
	
	if (&MethGet) {
		$variable = $ENV{'QUERY_STRING'};
	} elsif (&MethPost) {
		read(STDIN, $variable, $ENV{'CONTENT_LENGTH'});
	}
	
	@list = split(/[&;]/,$variable);
	
	foreach $i (0 .. $#list) {
		$list[$i] =~ s/\+/ /g;
		($key, $val) = split(/=/,$list[$i],2);
		$key =~ s/%(..)/pack("c",hex($1))/ge;
		$val =~ s/%(..)/pack("c",hex($1))/ge;
		$$varRef{$key} .= "\0" if (defined($$varRef{$key}));
		$$varRef{$key} .= $val;
	}
	return scalar($#list + 1);
}
sub MethGet { return ($ENV{'REQUEST_METHOD'} eq "GET") };
sub MethPost { return ($ENV{'REQUEST_METHOD'} eq "POST") };

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

# Just for testing
#==============================================================
sub dumpHash {
	foreach my $key (keys(%frmVals)) { print "$key, $frmVals{$key}; <BR>" };
}
